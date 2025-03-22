plugin.tar.gz: plugin.exe plugin.json
	tar -czvf plugin.tar.gz plugin.exe plugin.json

plugin.exe: *.go
	go build -o plugin.exe *.go