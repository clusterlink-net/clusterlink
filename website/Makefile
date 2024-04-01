#------------------------------------------------------
# website targets
#------------------------------------------------------
.PHONY: module_update build clean serve serve-dev checklinks checklinks-dev

module_update: ; $(info updating hugo modules...)
	@hugo mod get -u ./...
	@hugo mod tidy

build: ; $(info building site...)
	@hugo

# delete hugo modules under clean?
clean: ; $(info cleaning website output directory...)
	@rm -rf ./public

serve: ; $(info running locally with env=production...)
	@hugo server --gc --cleanDestinationDir --disableFastRender

serve-dev: ; $(info running locally env=dev \(drafts and future\)...)
	@hugo server --gc --cleanDestinationDir --disableFastRender --buildDrafts --buildFuture

checklinks: ; $(info running link check on https://clusterlink.net...)
	@muffet -c 16 -b 65536  --rate-limit 16 https://clusterlink.net

# run "make serve-dev" in one terminal and this command in another...
checklinks-dev: ; $(info running link check on http://localhost:1313...)
	@muffet -c 16 -b 65536  --rate-limit 16 http://localhost:1313
