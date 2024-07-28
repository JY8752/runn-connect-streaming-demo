.PHONY: books
books:
	@find books -type f -name "*.yaml" | sed 's|[^/]*$$|*.yaml|' | uniq | xargs -I {} runn run --scopes run:exec {} --concurrent on
