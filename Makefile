plugin.tar.gz: plugin.exe
	tar -czvf plugin.tar.gz plugin.exe plugin.json

plugin.exe: plugin.go
	go build -o plugin.exe plugin.go