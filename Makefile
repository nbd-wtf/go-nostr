### Tools needed for development
devtools:
	@echo "Installing devtools"
	go install mvdan.cc/gofumpt@latest

### Formatting, linting, and vetting
fmt:
	gofumpt -l -w .
