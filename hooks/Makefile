.PHONY: all clean build-ag-ui-emitter build-todo-monitor

all: build-ag-ui-emitter build-todo-monitor

build-ag-ui-emitter:
	@echo "Building ag-ui-emitter hook..."
	@cd ../cmd/hooks/ag-ui-emitter && go mod download && go build -o ../../../hooks/alpine-ag-ui-emitter .
	@chmod +x alpine-ag-ui-emitter
	@echo "Built alpine-ag-ui-emitter"

build-todo-monitor:
	@echo "Building todo-monitor hook..."
	@cd ../cmd/hooks/todo-monitor && go mod download && go build -o ../../../hooks/alpine-todo-monitor .
	@chmod +x alpine-todo-monitor
	@echo "Built alpine-todo-monitor"

clean:
	@rm -f alpine-ag-ui-emitter alpine-todo-monitor
	@echo "Cleaned hook binaries"