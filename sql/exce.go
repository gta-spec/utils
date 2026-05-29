package _sql

import (
	"database/sql"
	"fmt"
	"strings"
)

// Exec 分割并执行SQL文件中的语句（增强版）
func Exec(db *sql.DB, sqlContent string) error {
	// 移除注释
	sqlContent = removeSQLComments(sqlContent)

	// 按分号分割语句
	statements := splitSQLStatements(sqlContent)

	for _, statement := range statements {
		trimmedStmt := strings.TrimSpace(statement)
		if trimmedStmt == "" {
			continue
		}

		// 跳过某些系统命令
		if strings.HasPrefix(strings.ToUpper(trimmedStmt), "SET ") {
			continue
		}

		// 执行单条SQL语句
		_, err := db.Exec(trimmedStmt)
		if err != nil {
			return fmt.Errorf("error executing statement: %s, err: %v", trimmedStmt, err)
		}
	}

	return nil
}

// removeSQLComments 移除SQL注释
func removeSQLComments(sql string) string {
	for {
		start := strings.Index(sql, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(sql[start+2:], "*/")
		if end == -1 {
			break
		}
		sql = sql[:start] + sql[start+2+end+2:]
	}

	lines := strings.Split(sql, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过单行注释行
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// 移除行尾注释
		if strings.Contains(line, "--") {
			line = line[:strings.Index(line, "--")]
		}
		if strings.Contains(line, "#") {
			line = line[:strings.Index(line, "#")]
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// splitSQLStatements 分割SQL语句
func splitSQLStatements(sql string) []string {
	var statements []string
	var currentStmt strings.Builder

	inString := false
	escapeNext := false
	quoteChar := rune(0)

	for _, char := range sql {
		if escapeNext {
			currentStmt.WriteRune(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			currentStmt.WriteRune(char)
			escapeNext = true
			continue
		}

		if !inString && (char == '"' || char == '\'' || char == '`') {
			inString = true
			quoteChar = char
			currentStmt.WriteRune(char)
			continue
		}

		if inString && char == quoteChar {
			inString = false
			currentStmt.WriteRune(char)
			continue
		}

		if !inString && char == ';' {
			statements = append(statements, currentStmt.String())
			currentStmt.Reset()
			continue
		}

		currentStmt.WriteRune(char)
	}

	// 添加最后一条语句（如果没有以分号结尾）
	if currentStmt.Len() > 0 {
		lastStmt := strings.TrimSpace(currentStmt.String())
		if lastStmt != "" {
			statements = append(statements, lastStmt)
		}
	}

	return statements
}
