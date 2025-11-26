package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"time"

	"dify-cli-gateway/internal/release_notes"
)

func main() {
	log.Println("🚀 Starting release notes HTML generator...")

	// 创建服务配置
	config := release_notes.ServiceConfig{
		CacheTTL:        time.Hour,
		RefreshInterval: time.Hour,
		StoragePath:     "data/release_notes.json",
	}

	// 创建服务
	service := release_notes.NewReleaseNotesService(config)
	service.InitializeFetchers()

	// 获取所有 release notes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("📥 Fetching release notes for all CLIs...")
	if err := service.Refresh(ctx, true); err != nil {
		log.Printf("⚠️ Warning: Some fetches failed: %v", err)
	}

	allNotes, err := service.GetAll(ctx, false)
	if err != nil {
		log.Fatalf("❌ Failed to get release notes: %v", err)
	}

	log.Printf("✅ Fetched release notes for %d CLIs", len(allNotes.CLIs))

	// 读取模板
	templatePath := filepath.Join("web", "templates", "release_notes_static.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("❌ Failed to parse template: %v", err)
	}

	// 生成 HTML
	outputPath := "release-notes.html"
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("❌ Failed to create output file: %v", err)
	}
	defer f.Close()

	// 准备模板数据
	data := struct {
		AllNotes    *release_notes.AllReleaseNotes
		GeneratedAt string
	}{
		AllNotes:    allNotes,
		GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	}

	if err := tmpl.Execute(f, data); err != nil {
		log.Fatalf("❌ Failed to execute template: %v", err)
	}

	log.Printf("✅ HTML generated successfully: %s", outputPath)
	fmt.Printf("\n🎉 Release notes HTML has been generated!\n")
	fmt.Printf("📄 Output file: %s\n", outputPath)
	fmt.Printf("📊 Total CLIs: %d\n", len(allNotes.CLIs))
	fmt.Printf("🕐 Generated at: %s\n", data.GeneratedAt)
}
