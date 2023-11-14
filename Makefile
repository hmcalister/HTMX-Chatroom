dev: ; make -j2 devCSS devAIR

devCSS:
	cd static/css; npx tailwindcss -i input.css -o output.css --watch 

devAIR:
	air -tmp_dir "airtmp" --build.cmd "go build -o airtmp/main" --build.bin "airtmp/main" --build.exclude_dir "static/css" --build.send_interrupt True
	
build:
	cd static/css; npx tailwindcss -i input.css -o output.css --minify
	go build .
