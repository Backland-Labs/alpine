PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
CLI_NAME := sprite-opencode

.PHONY: help install uninstall

help:
	@printf 'Targets:\n'
	@printf '  make install   Install $(CLI_NAME) into $(BINDIR)\n'
	@printf '  make uninstall Remove $(BINDIR)/$(CLI_NAME)\n'

install:
	@mkdir -p "$(BINDIR)"
	@ln -sfn "$(CURDIR)/$(CLI_NAME)" "$(BINDIR)/$(CLI_NAME)"
	@printf 'Installed %s -> %s\n' "$(BINDIR)/$(CLI_NAME)" "$(CURDIR)/$(CLI_NAME)"

uninstall:
	@rm -f "$(BINDIR)/$(CLI_NAME)"
	@printf 'Removed %s\n' "$(BINDIR)/$(CLI_NAME)"
