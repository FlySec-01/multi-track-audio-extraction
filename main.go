package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// 定义一个结构体来匹配 JSON 列表中的每个元素
type Item struct {
	Rownum   int    `json:"rownum"`
	TypeName string `json:"typeName"`
}

// 获取当前目录中的所有 MP4 文件
func getMP4Files() ([]string, error) {
	var files []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".mp4") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func executeCommand(command string, args ...string) error {
	fmt.Println("执行命令:", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func zipFiles(zipFileName string, files []string) error {
	newZipFile, err := os.Create(zipFileName)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	for _, file := range files {
		if err := addFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Method = zip.Deflate
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, fileToZip)
	return err
}

func waitForExit() {
	fmt.Println("按任意键退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func main() {
	// 获取当前目录中的 MP4 文件
	files, err := getMP4Files()
	if err != nil {
		log.Fatalf("获取 MP4 文件出错: %v", err)
	}

	if len(files) == 0 {
		fmt.Println("当前目录中没有找到 MP4 文件。")
		waitForExit()
		return
	}

	// 显示文件列表供用户选择
	fmt.Println("请选择一个视频文件:")
	for i, file := range files {
		fmt.Printf("[%d] %s\n", i+1, file)
	}

	// 读取用户选择
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入视频文件的编号: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(files) {
		fmt.Println("选择无效。")
		waitForExit()
		return
	}

	videoResource := files[index-1]
	videoName := strings.TrimSuffix(filepath.Base(videoResource), filepath.Ext(videoResource))
	outputDir := videoName

	// 确保输出目录存在
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.Mkdir(outputDir, 0755)
		if err != nil {
			log.Fatalf("创建输出目录出错: %v", err)
		}
	}

	// 读取用户输入的 JSON 数据
	fmt.Print("请输入 JSON 数据: ")
	jsonInput, _ := reader.ReadString('\n')
	jsonInput = strings.TrimSpace(jsonInput)

	// 将单引号替换为双引号
	jsonInput = strings.ReplaceAll(jsonInput, "'", "\"")

	// 定义一个 Item 结构体的切片来保存解析结果
	var items []Item

	// 解析 JSON 数据
	err = json.Unmarshal([]byte(jsonInput), &items)
	if err != nil {
		fmt.Println("解析 JSON 出错，按任意键退出...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		return
		// log.Fatalf(": %v", err)
	}

	// 预处理 JSON 数据
	sort.Slice(items, func(i, j int) bool {
		return items[i].Rownum < items[j].Rownum
	})

	// 生成并执行第一个额外的 ffmpeg 命令
	extraOutputFile := filepath.Join(outputDir, "bz.mp3")
	extraCommand := []string{"-i", videoResource, "-map", "0:1", "-b:a", "129k", "-f", "mp3", "-vn", extraOutputFile}
	if err := executeCommand("./ffmpeg.exe", extraCommand...); err != nil {
		log.Fatalf("执行命令出错: %v", err)
	}

	// 保存所有生成的 mp3 文件的名称
	var mp3Files []string
	mp3Files = append(mp3Files, extraOutputFile)

	// 遍历解析后的数据并生成并执行 ffmpeg 命令
	for _, item := range items {
		audioTrackID := item.Rownum + 1
		audioTrackName := item.TypeName
		outputFileName := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", audioTrackName))
		command := []string{"-i", videoResource, "-map", fmt.Sprintf("0:%d", audioTrackID), "-b:a", "129k", "-f", "mp3", "-vn", outputFileName}
		if err := executeCommand("./ffmpeg.exe", command...); err != nil {
			log.Fatalf("执行命令出错: %v", err)
		}
		mp3Files = append(mp3Files, outputFileName)
	}

	// 打包所有生成的 mp3 文件到新目录下
	zipFileName := filepath.Join(outputDir, fmt.Sprintf("%s_output.zip", videoName))
	if err := zipFiles(zipFileName, mp3Files); err != nil {
		log.Fatalf("打包文件出错: %v", err)
	}
	fmt.Printf("所有 MP3 文件已打包成 %s,按任意键退出\n", zipFileName)
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	return
}
