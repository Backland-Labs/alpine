.PHONY: build clean install test

build:
	npm run build

clean:
	npm run clean

test:
	npm test

install: build
	npm link
	@printf "installed sc via npm link\n"
