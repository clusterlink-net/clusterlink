#------------------------------------------------------
# website targets
#------------------------------------------------------
.PHONY: module_update build clean serve serve-dev

module_update:
	@hugo mod clean
	@hugo mod get

build:
	@hugo

# delete hugo modules under clean?
clean:
	@rm -rf ./public

serve:
	@hugo server --gc --cleanDestinationDir --disableFastRender

serve-dev:
	@hugo server --gc --cleanDestinationDir --disableFastRender --buildDrafts --buildFuture
