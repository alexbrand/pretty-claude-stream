.PHONY: build install clean

build:
	go build -o pretty-claude-stream .

install:
	go install .

clean:
	rm -f pretty-claude-stream
