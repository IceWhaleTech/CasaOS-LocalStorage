#!/usr/bin/bash

rm -rf target
mkdir target
docker run --rm --user "$(id -u):$(id -g)" -v "${PWD}":/local -e GO_POST_PROCESS_FILE="bash /local/openapi-generator-postprocess.sh" openapitools/openapi-generator-cli generate -i /local/openapi.yaml -g go-server -o /local/target -p=onlyInterfaces=true,outputAsLibrary=true,router=chi,featureCORS=true,sourceFolder=codegen --strict-spec true --enable-post-process-file
