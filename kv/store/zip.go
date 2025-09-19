package store

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// createZip 创建一个ZIP归档文件
// srcDir: 要压缩的源目录
// dstFile: 目标ZIP文件路径
func createZip(srcDir string, dstFile string) error {
	// 创建目标ZIP文件
	zipFile, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// 创建zip writer
	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	// 遍历源目录
	return filepath.Walk(srcDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过根目录
		relPath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// 为目录添加斜杠后缀
		if info.IsDir() {
			relPath = relPath + "/"
		}

		// 创建zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate // 使用DEFLATE算法压缩

		// 写入header
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		// 如果是普通文件则写入内容
		if info.Mode().IsRegular() {
			f, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(writer, f); err != nil {
				return err
			}
		}
		return nil
	})
}

// extractZip 解压ZIP文件到指定目录
// srcFile: 源ZIP文件路径
// destDir: 目标解压目录
func extractZip(srcFile string, destDir string) error {
	// 打开ZIP文件
	reader, err := zip.OpenReader(srcFile)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 获取目标目录绝对路径
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}

	// 创建目标目录
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// 遍历所有文件
	for _, file := range reader.File {
		// 验证目标路径安全
		targetPath := filepath.Join(destDir, file.Name)
		targetAbs, err := filepath.Abs(targetPath)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(targetAbs, destAbs) {
			return fmt.Errorf("unsafe extraction path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			// 处理目录
			if err := handleZipDirectory(targetPath, file); err != nil {
				return err
			}
		} else {
			// 处理普通文件
			if err := handleZipFile(targetPath, file); err != nil {
				return err
			}
		}
	}
	return nil
}

// handleZipDirectory 处理ZIP归档中的目录项
func handleZipDirectory(targetPath string, file *zip.File) error {
	if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
		return err
	}
	// 设置目录权限
	if err := os.Chmod(targetPath, file.Mode()); err != nil {
		return err
	}
	// 设置时间
	return os.Chtimes(targetPath, time.Now(), file.Modified)
}

// handleZipFile 处理ZIP归档中的普通文件
func handleZipFile(targetPath string, file *zip.File) error {
	// 创建父目录
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// 打开归档中的文件
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 创建目标文件
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
	if err != nil {
		return err
	}
	defer f.Close()

	// 写入内容
	if _, err := io.Copy(f, rc); err != nil {
		return err
	}

	// 设置权限（覆盖umask的影响）
	if err := os.Chmod(targetPath, file.Mode()); err != nil {
		return err
	}

	// 设置时间
	return os.Chtimes(targetPath, time.Now(), file.Modified)
}
