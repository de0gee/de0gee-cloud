
buildyarn:
	yarn install
	yarn build
	cp build/index.html ../../templates/realtime.tmpl
	rm -rf ../../static/js/main.*.*
	rm -rf ../../static/css/main.*.*
	rsync -avrP build/static/ ../../static/
	git add ../../static/										
