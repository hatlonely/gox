package store

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func createTarGz(srcDir string, dstFile string) error {
	// 创建目标文件
	fw, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer fw.Close()

	// 创建gzip writer
	gw := gzip.NewWriter(fw)
	defer gw.Close()

	// 创建tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

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

		// 创建tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// 写入header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// 如果是普通文件则写入内容
		if info.Mode().IsRegular() {
			f, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})
}

func extractTarGz(srcFile string, destDir string) error {

	// 打开压缩文件
	fr, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer fr.Close()

	// 创建gzip reader
	gr, err := gzip.NewReader(fr)
	if err != nil {
		return err
	}
	defer gr.Close()

	// 创建tar reader
	tr := tar.NewReader(gr)

	// 获取目标目录绝对路径
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 验证目标路径安全
		targetPath := filepath.Join(destDir, header.Name)
		targetAbs, err := filepath.Abs(targetPath)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(targetAbs, destAbs) {
			return fmt.Errorf("unsafe extraction path: %s", header.Name)
		}

		// 处理不同类型文件
		switch header.Typeflag {
		case tar.TypeDir:
			if err := handleDirectory(targetPath, header); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := handleRegularFile(targetPath, tr, header); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported file type: %v in %s", header.Typeflag, header.Name)
		}
	}

	return nil
}

func handleDirectory(targetPath string, header *tar.Header) error {
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}
	// 设置目录权限
	if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
		return err
	}
	// 设置时间
	return os.Chtimes(targetPath, time.Now(), header.ModTime)
}

func handleRegularFile(targetPath string, tr *tar.Reader, header *tar.Header) error {
	// 创建父目录
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// 创建文件
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
	if err != nil {
		return err
	}
	defer f.Close()

	// 写入内容
	if _, err := io.Copy(f, tr); err != nil {
		return err
	}

	// 设置权限（覆盖umask的影响）
	if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
		return err
	}

	// 设置时间
	return os.Chtimes(targetPath, time.Now(), header.ModTime)
}
