SHELL := /bin/bash

.PHONY: docs fmt-md lint-md

MD_FILES := $(shell ls -1 *.md docs/*.md 2>/dev/null || true)

docs: fmt-md lint-md

fmt-md:
	@echo "Formatting Markdown (trim trailing spaces; ensure newline EOF)"
	@for f in $(MD_FILES); do \
		perl -0777 -pe 's/[ \t]+\n/\n/g; END{print "\n" unless /\n\z/}' $$f > $$f.tmp && mv $$f.tmp $$f; \
	done

lint-md:
	@bash scripts/md_style_check.sh

