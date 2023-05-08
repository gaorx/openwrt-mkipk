package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	isHelp         bool
	inputDir       string
	outputFilename string
)

func init() {
	flag.BoolVar(&isHelp, "h", false, "Show this help")
	flag.StringVar(&inputDir, "i", "", "Input directory path")
	flag.StringVar(&outputFilename, "o", "", "Output IPK dir/file")
}

func main() {
	flag.Parse()
	if isHelp {
		printUsage()
		return
	}

	// 检测参数
	if inputDir == "" {
		fmt.Println("no input directory")
		os.Exit(1)
	}
	if outputFilename == "" {
		fmt.Println("no output ipk filename")
		os.Exit(1)
	}
	if !isDir(inputDir) {
		fmt.Printf("illegal input directory '%s'\n", inputDir)
		os.Exit(1)
	}
	controlDir := filepath.Join(inputDir, "control")
	dataDir := filepath.Join(inputDir, "data")
	if !isDir(controlDir) {
		fmt.Printf("illegal input control dir '%s'\n", controlDir)
		os.Exit(1)
	}
	if !isDir(dataDir) {
		fmt.Printf("illegal input data dir '%s'\n", dataDir)
		os.Exit(1)
	}

	// 压缩control.tar.gz
	controlGzipBuff, err := gzipDir(controlDir, gzipFileOptions{
		User:  "root",
		Group: "root",
	})
	if err != nil {
		fmt.Printf("gzip error '%s'\n", controlDir)
		os.Exit(1)
	}

	// 压缩data.tar.gz
	dataGzipBuff, err := gzipDir(dataDir, gzipFileOptions{
		User:  "root",
		Group: "root",
	})
	if err != nil {
		fmt.Printf("gzip error '%s'\n", dataDir)
		os.Exit(1)
	}

	// 将control.tar.gz 和 data.tar.gz 和 debian-binary写入临时目录
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("mkipk_%d", time.Now().UnixMilli()))
	err = os.MkdirAll(tmpDir, 0755)
	if err != nil {
		fmt.Printf("create tmp dir error '%s'\n", tmpDir)
		os.Exit(1)
	}
	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()
	err = writeBytesTo(controlGzipBuff, filepath.Join(tmpDir, "control.tar.gz"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = writeBytesTo(dataGzipBuff, filepath.Join(tmpDir, "data.tar.gz"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = writeBytesTo([]byte("2.0"), filepath.Join(tmpDir, "debian-binary"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 将临时目录压缩到ipk
	targetBuff, err := gzipDir(tmpDir, gzipFileOptions{
		User:  "root",
		Group: "root",
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 将ipk写入到文件中
	if isDir(outputFilename) {
		c, err := readControl(inputDir)
		if err != nil {
			fmt.Printf("read control error '%s'\n", inputDir)
			os.Exit(1)
		}
		var basename string
		if c.Arch != "" {
			basename = fmt.Sprintf("%s_%s_%s.ipk", c.Package, c.Version, c.Arch)
		} else {
			basename = fmt.Sprintf("%s_%s.ipk", c.Package, c.Version)
		}
		outputFilename = filepath.Join(outputFilename, basename)
	}
	err = writeBytesTo(targetBuff, outputFilename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("OK '%s'\n", outputFilename)
}

type gzipFileOptions struct {
	User  string
	Group string
	Uid   int
	Gid   int
}

func gzipDir(dirname string, fileOpts gzipFileOptions) ([]byte, error) {
	absDirname, err := filepath.Abs(dirname)
	if err != nil {
		return nil, err
	}
	var filelist []string
	err = filepath.Walk(absDirname, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == absDirname {
			return nil
		}
		rel, err := filepath.Rel(absDirname, path)
		if err != nil {
			return err
		}
		filelist = append(filelist, "./"+rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(filelist) <= 0 {
		return nil, nil
	}
	rawBuff := bytes.NewBuffer(nil)
	tw := tar.NewWriter(rawBuff)
	for _, fn := range filelist {
		absFn := filepath.Join(absDirname, fn)
		fi, err := os.Stat(absFn)
		if err != nil {
			return nil, err
		}
		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return nil, err
		}
		hdr.Name = fn
		hdr.Uid, hdr.Uname = fileOpts.Uid, fileOpts.User
		hdr.Gid, hdr.Gname = fileOpts.Gid, fileOpts.Group
		err = tw.WriteHeader(hdr)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			data, err := readToBytes(absFn)
			if err != nil {
				return nil, err
			}
			nCopy, err := io.Copy(tw, bytes.NewBuffer(data))
			if err != nil {
				return nil, err
			}
			if nCopy != int64(len(data)) {
				return nil, err
			}
		}
		_ = tw.Flush()
	}
	_ = tw.Close()

	zipBuff := bytes.NewBuffer(nil)
	zw := gzip.NewWriter(zipBuff)
	zw.Header.OS = 0x03 // Unix
	_, err = zw.Write(rawBuff.Bytes())
	if err != nil {
		return nil, err
	}
	_ = zw.Flush()
	_ = zw.Close()
	return zipBuff.Bytes(), nil
}

func printUsage() {
	logo := `mkipk - Openwrt ipk tool (by GaoRongxin)`
	tree := `/path/to/input_dir
  ├── control
  │   ├── control
  │   └── postinst
  └── data
      ├── app
      ├── etc
      └── usr
           └── lib
	`
	fmt.Println(logo)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Usage:")
	fmt.Println("  go run mkipk.go -i /path/to/input_dir -o /path/to/(output_dir|output_file)")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Input directory layout:")
	fmt.Println(tree)
}

func isDir(dirname string) bool {
	fi, err := os.Stat(dirname)
	if err != nil {
		return false
	}
	if !fi.IsDir() {
		return false
	}
	return true
}

func readToBytes(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeBytesTo(data []byte, filename string) error {
	return os.WriteFile(filename, data, 0644)
}

type control struct {
	Package string
	Version string
	Arch    string
}

func readControl(dirname string) (*control, error) {
	data, err := readToBytes(filepath.Join(dirname, "control", "control"))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var r control
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		segments := strings.SplitN(line, ":", 2)
		if len(segments) != 2 {
			continue
		}
		k, v := strings.TrimSpace(segments[0]), strings.TrimSpace(segments[1])
		switch strings.ToLower(k) {
		case "package":
			r.Package = v
		case "version":
			r.Version = v
		case "architecture":
			r.Arch = v
		}
	}
	if r.Package == "" {
		return nil, errors.New("no Package field")
	}
	if r.Version == "" {
		return nil, errors.New("no Version field")
	}
	return &r, nil
}
