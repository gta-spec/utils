package _slog

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	
	"github.com/gta-spec/utils/time"
)

type Reason string

const (
	backupTimeFormat        = "2006-01-02T15-04-05.000"
	compressSuffix          = ".gz"
	defaultMaxSize          = 100
	ReasonSize       Reason = "size"
	ReasonTime       Reason = "time"
)

type constError string

func (c constError) Error() string {
	return string(c)
}

// ErrWriteTooLong indicates that a single write that is longer than the max
// size allowed in a single file.
const ErrWriteTooLong = constError("write exceeds max file length")

// Options represents optional behavior you can specify for a new Roller.
type Options struct {
	// MaxAge is the maximum time to retain old log files based on the timestamp
	// encoded in their filename. The default is not to remove old log files
	// based on age.
	MaxAge time.Duration
	
	// MaxBackups is the maximum number of old log files to retain. The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int
	
	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time. The default is to use UTC
	// time.
	LocalTime bool
	
	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool
}

// NewRoller returns a new Roller.
//
// If the file exists and is less than maxSize bytes, lumberjack will open and
// append to that file. If the file exists and its size is >= maxSize bytes, the
// file is renamed by putting the current time in a timestamp in the name
// immediately before the file's extension (or the end of the filename if
// there's no extension). A new log file is then created using original
// filename.
//
// An error is returned if a file cannot be opened or created, or if maxsize is
// 0 or less.
func NewRoller(filename string, maxSize int64, opt *Options) (*Roller, error) {
	if maxSize <= 0 {
		return nil, errors.New("max size cannot be 0")
	}
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}
	
	filename = filepath.FromSlash(filename)
	// 移除文件后缀名 , 只要在创建文件的时候加上
	ext := filepath.Ext(filename)
	
	var baseDir string
	
	// 匹配第一个 strftime 之前的路径
	loc := _time.StrftimePattern.FindStringIndex(filename)
	if loc == nil {
		baseDir = filepath.Dir(filename)
	} else {
		prefix := filename[:loc[0]]
		baseDir = filepath.Dir(prefix)
	}
	
	r := &Roller{
		filename: strings.TrimSuffix(filename, ext),
		maxSize:  maxSize,
		ext:      ext,
		baseDir:  baseDir,
	}
	
	if strings.HasSuffix(r.filename, ".0") {
		r.filename = strings.TrimSuffix(r.filename, ".0")
		r.withZeroIdx = true
	}
	
	if opt != nil {
		r.maxAge = opt.MaxAge
		r.maxBackups = opt.MaxBackups
		r.localTime = opt.LocalTime
		r.compress = opt.Compress
	}
	
	// 判断filename是否随时间轮转
	if _time.IsStrftimeFormat(r.filename) {
		r.strftime = _time.Strftime(r.filename)
		t := currentTime()
		if !r.localTime {
			t = t.UTC()
		}
		r.filename = t.Format(r.strftime)
	}
	
	if oldLogFiles, err := r.oldLogFiles(); err == nil && len(oldLogFiles) > 0 {
		r.filename = strings.TrimSuffix(oldLogFiles[0].name, r.ext)
	}
	
	err := r.openExistingOrNew(0)
	if err != nil {
		return nil, fmt.Errorf("can't open file: %w", err)
	}
	return r, nil
}

// Roller wraps a file, intercepting its writes to control its size, rolling the
// old file over to a different name before writing to a new one.
//
// Whenever a write would cause the current log file exceed maxSize bytes, the
// current file is closed, renamed, and a new log file created with the original
// name. Thus, the filename you give Roller is always the "current" log file.
//
// Backups use the log file name given to Roller, in the form
// `name-timestamp.ext` where name is the filename without the extension,
// timestamp is the time at which the log was rotated formatted with the
// time.Time format of `2006-01-02T15-04-05.000` and the extension is the
// original extension. For example, if your Roller.Filename is
// `/var/log/foo/server.log`, a backup created at 6:30pm on Nov 11 2016 would
// use the filename `/var/log/foo/server-2016-11-04T18-30-00.000.log`
//
// # Cleaning Up Old Log Files
//
// Whenever a new logfile gets created, old log files may be deleted. The most
// recent files according to the encoded timestamp will be retained, up to a
// number equal to MaxBackups (or all of them if MaxBackups is 0). Any files
// with an encoded timestamp older than MaxAge days are deleted, regardless of
// MaxBackups. Note that the time encoded in the timestamp is the rotation
// time, which may differ from the last time that file was written to.
//
// If MaxBackups and MaxAge are both 0, no old log files will be deleted.
type Roller struct {
	// filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	filename string
	
	// maxSize is the maximum size in bytes of the log file before it gets
	// rotated.
	maxSize int64
	
	// maxAge is the maximum time to retain old log files based on the timestamp
	// encoded in their filename. The default is not to remove old log files
	// based on age.
	maxAge time.Duration
	
	// maxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	maxBackups int
	
	// localTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	localTime bool
	
	// compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	compress bool
	
	size int64
	file *os.File
	mu   sync.Mutex
	
	millCh    chan bool
	startMill sync.Once
	
	strftime    string
	baseDir     string //第一个 %时间之前的路径
	ext         string
	withZeroIdx bool
}

var (
	// currentTime exists so it can be mocked out by tests.
	currentTime = time.Now
	
	// os_Stat exists so it can be mocked out by tests.
	osStat = os.Stat
)

// Write implements io.Writer.  If a write would cause the log file to be larger
// than MaxSize, the file is closed, renamed to include a timestamp of the
// current time, and a new log file is created using the original log file name.
// If the length of the write is greater than MaxSize, an error is returned.
func (r *Roller) Write(p []byte) (n int, err error) {
	writeLen := int64(len(p))
	if writeLen > r.maxSize {
		return 0, fmt.Errorf(
			"write length %d, max size %d: %w", writeLen, r.maxSize, ErrWriteTooLong,
		)
	}
	
	// 预计算时间相关变量（锁外操作，减少锁持有时间）
	var (
		now         time.Time
		newFilename string
	)
	if r.strftime != "" {
		now = currentTime()
		timeLayout := now
		if !r.localTime {
			timeLayout = timeLayout.UTC()
		}
		// 缓存格式化结果，避免多次计算
		newFilename = timeLayout.Format(r.strftime)
	}
	
	defer r.mu.Unlock()
	r.mu.Lock()
	
	// 时间轮转判断（优化路径拼接和解析逻辑）
	if r.strftime != "" {
		// 解析当前文件名的路径信息
		oldDir, oldBase, _ := pathInfo(r.filename)
		oldFilename := filepath.Join(oldDir, oldBase)
		
		// 快速判断：文件名未变化则无需轮转
		if newFilename != oldFilename {
			if err := r.rotate(ReasonTime); err != nil {
				return 0, err
			}
		}
	}
	
	// 按照文件大小轮转
	if r.size+writeLen > r.maxSize {
		if err := r.rotate(ReasonSize); err != nil {
			return 0, err
		}
	}
	
	n, err = r.file.Write(p)
	r.size += int64(n)
	
	return n, err
}

// Close implements io.Closer, and closes the current logfile.
func (r *Roller) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.close()
}

// close closes the file if it is open.
func (r *Roller) close() error {
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

// Rotate causes Logger to close the existing log file and immediately create a
// new one.  This is a helper function for applications that want to initiate
// rotations outside of the normal rotation rules, such as in response to
// SIGHUP.  After rotating, this initiates compression and removal of old log
// files according to the configuration.
func (r *Roller) Rotate() error {
	defer r.mu.Unlock()
	r.mu.Lock()
	return r.rotate(ReasonSize)
}

// rotate closes the current file, moves it aside with a timestamp in the name,
// (if it exists), opens a new file with the original filename, and then runs
// post-rotation processing and removar.
func (r *Roller) rotate(reason Reason) error {
	if err := r.close(); err != nil {
		return err
	}
	switch reason {
	case ReasonTime:
		t := currentTime()
		if !r.localTime {
			t = t.UTC()
		}
		r.filename = t.Format(r.strftime)
	case ReasonSize:
		d, b, i := pathInfo(r.filename)
		base := filepath.Join(d, b)
		if r.withZeroIdx && i == 0 {
			of := base + r.ext
			nf := base + ".0" + r.ext
			_ = os.Rename(of, nf)
		}
		r.filename = base + "." + strconv.Itoa(i+1)
	}
	if err := r.openNew(); err != nil {
		return err
	}
	r.mill()
	return nil
}

// openNew opens a new log file for writing, moving any old log file out of the
// way.  This methods assumes the file has already been closed.
func (r *Roller) openNew() error {
	name := r.newFilename() + r.ext
	err := os.MkdirAll(filepath.Dir(name), 0755)
	
	if err != nil {
		return fmt.Errorf("can't make directories for new logfile: %w", err)
	}
	
	mode := os.FileMode(0600)
	info, err := osStat(name)
	if err == nil {
		// Copy the mode off the old logfile.
		mode = info.Mode()
		
		// this is a no-op anywhere but linux
		if err := chown(name, info); err != nil {
			return err
		}
	}
	
	// we use truncate here because this should only get called when we've moved
	// the file ourselves. if someone else creates the file in the meantime,
	// just wipe out the contents.
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %w", err)
	}
	r.file = f
	r.size = 0
	return nil
}

// openExistingOrNew opens the logfile if it exists and if the current write
// would not put it over MaxSize.  If there is no such file or the write would
// put it over the MaxSize, a new file is created.
func (r *Roller) openExistingOrNew(writeLen int64) error {
	r.mill()
	
	filename := r.newFilename() + r.ext
	info, err := osStat(filename)
	if os.IsNotExist(err) {
		return r.openNew()
	}
	
	if err != nil {
		return fmt.Errorf("error getting log file info: %w", err)
	}
	
	if r.strftime != "" {
		d, b, _ := pathInfo(r.filename)
		bf := filepath.Join(d, b)
		if bf != currentTime().Format(r.strftime) {
			return r.rotate(ReasonTime)
		}
	}
	
	if info.Size()+writeLen >= r.maxSize {
		return r.rotate(ReasonSize)
	}
	
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// if we fail to open the old log file for some reason, just ignore
		// it and open a new log file.
		return r.openNew()
	}
	r.file = file
	r.size = info.Size()
	return nil
}

// newFilename generates the name of the logfile from the current time.
func (r *Roller) newFilename() string {
	if r.filename != "" {
		return r.filename
	}
	name := filepath.Base(os.Args[0]) + "-lumberjack.log"
	return filepath.Join(os.TempDir(), name)
}

// pathInfo (已经移除后缀)解析日志文件路径信息，提取目录、文件名、前缀、扩展名和轮转索引。
//
// 返回值说明：
//
//	 例如:    "runtime/log/2026-05-15.1"
//		dir:     文件所在目录路径（如 "/runtime/log"）
//		base:    完整文件名（含扩展名和索引，如 "2026-05-15"）
//		index:   轮转索引号（无索引时为 0，如 1）
func pathInfo(filename string) (string, string, int) {
	var index int
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	
	iExt := filepath.Ext(base)
	
	if iExt != "" {
		if i, err := strconv.Atoi(iExt[1:]); err == nil {
			base = strings.TrimSuffix(base, iExt)
			index = i
		}
	}
	return dir, base, index
}

// millRunOnce performs compression and removal of stale log files.
// Log files are compressed if enabled via configuration and old log
// files are removed, keeping at most r.MaxBackups files, as long as
// none of them are older than MaxAge.
func (r *Roller) millRunOnce() error {
	if r.maxBackups == 0 && r.maxAge == 0 && !r.compress {
		return nil
	}
	
	files, err := r.oldLogFiles()
	if err != nil {
		return err
	}
	
	var compress, remove []logInfo
	
	if r.maxBackups > 0 && r.maxBackups < len(files) {
		preserved := make(map[string]bool)
		var remaining []logInfo
		for _, f := range files {
			// Only count the uncompressed log file or the
			// compressed log file, not both.
			fn := f.Name()
			if strings.HasSuffix(fn, compressSuffix) {
				fn = fn[:len(fn)-len(compressSuffix)]
			}
			preserved[fn] = true
			
			if len(preserved) > r.maxBackups {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}
	if r.maxAge > 0 {
		cutoff := currentTime().Add(-1 * r.maxAge)
		
		var remaining []logInfo
		for _, f := range files {
			if f.ModTime().Before(cutoff) {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}
	
	if r.compress {
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), compressSuffix) {
				compress = append(compress, f)
			}
		}
	}
	
	for _, f := range remove {
		errRemove := os.Remove(filepath.Join(r.dir(), f.Name()))
		if err == nil && errRemove != nil {
			err = errRemove
		}
	}
	for _, f := range compress {
		fn := filepath.Join(r.dir(), f.Name())
		errCompress := compressLogFile(fn, fn+compressSuffix)
		if err == nil && errCompress != nil {
			err = errCompress
		}
	}
	
	return err
}

// millRun runs in a goroutine to manage post-rotation compression and removal
// of old log files.
func (r *Roller) millRun() {
	for range r.millCh {
		// what am I going to do, log this?
		_ = r.millRunOnce()
	}
}

// mill performs post-rotation compression and removal of stale log files,
// starting the mill goroutine if necessary.
func (r *Roller) mill() {
	r.startMill.Do(func() {
		r.millCh = make(chan bool, 1)
		go r.millRun()
	})
	select {
	case r.millCh <- true:
	default:
	}
}

// oldLogFiles returns the list of backup log files stored in the same
// directory as the current log file, sorted by ModTime
func (r *Roller) oldLogFiles() ([]logInfo, error) {
	var logFiles []logInfo
	
	err := walkDir(r.baseDir, func(path string, entry os.DirEntry) {
		var timestamp time.Time
		
		fn := filepath.Join(path, entry.Name())
		if r.compress {
			fn = strings.TrimSuffix(fn, compressSuffix)
		}
		
		fn = strings.TrimSuffix(fn, r.ext)
		
		d, b, i := pathInfo(fn)
		if r.strftime != "" {
			t, err := time.Parse(r.strftime, filepath.Join(d, b))
			if err == nil {
				timestamp = t
			}
		}
		info, _ := entry.Info()
		logFiles = append(logFiles, logInfo{fn, i, timestamp, info})
	})
	
	if err != nil {
		return logFiles, nil
	}
	
	if len(logFiles) == 0 {
		return nil, fmt.Errorf("can't read log file directory: %w", err)
	}
	
	sort.Sort(byFormatTime(logFiles))
	
	return logFiles, nil
}

func walkDir(dir string, callback func(path string, entry os.DirEntry)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := walkDir(filepath.Join(dir, entry.Name()), callback); err != nil {
				return err
			}
		} else {
			callback(dir, entry)
		}
	}
	return nil
}

// dir returns the directory for the current filename.
func (r *Roller) dir() string {
	return filepath.Dir(r.newFilename())
}

// compressLogFile compresses the given log file, removing the
// uncompressed log file if successfur.
func compressLogFile(src, dst string) (err error) {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer f.Close()
	
	fi, err := osStat(src)
	if err != nil {
		return fmt.Errorf("failed to stat log file: %v", err)
	}
	
	if err := chown(dst, fi); err != nil {
		return fmt.Errorf("failed to chown compressed log file: %v", err)
	}
	
	// If this file already exists, we presume it was created by
	// a previous attempt to compress the log file.
	gzf, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fi.Mode())
	if err != nil {
		return fmt.Errorf("failed to open compressed log file: %v", err)
	}
	defer gzf.Close()
	
	gz := gzip.NewWriter(gzf)
	
	defer func() {
		if err != nil {
			os.Remove(dst)
			err = fmt.Errorf("failed to compress log file: %v", err)
		}
	}()
	
	if _, err := io.Copy(gz, f); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := gzf.Close(); err != nil {
		return err
	}
	
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return err
	}
	
	return nil
}

// logInfo is a convenience struct to return the filename and its embedded
// timestamp.
type logInfo struct {
	name      string
	index     int
	timestamp time.Time
	os.FileInfo
}

// byFormatTime sorts by newest time formatted in the name.
type byFormatTime []logInfo

func (b byFormatTime) Less(i, j int) bool {
	if b[i].timestamp.Equal(b[j].timestamp) {
		return b[i].index > b[j].index
	}
	return b[i].timestamp.After(b[j].timestamp)
}

func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byFormatTime) Len() int {
	return len(b)
}
