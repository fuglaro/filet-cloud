
all: filet-cloud-web npm/node_modules

filet-cloud-web: main.go
	go build

npm/node_modules: npm/package.json npm/package-lock.json
	cd npm; npm install; cd -

clean:
	rm -f filet-cloud-web
	rm -rf npm/node_modules

.PHONY: all clean
