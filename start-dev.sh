#!/usr/bin/env bash
CompileDaemon -build="go build" -include="*.tpl" -include="*.tmpl" -include="*.gohtml" -include="*.css" -recursive="true" -command="./zunkasrv dev"

# CompileDaemon \
# -build="go build" \
# -include="*.tpl" \
# -recursive="true" \
# -command="./bluewhale dev"

# -build="go build bluewhale.go handlers.go handlersInfo.go handlersAuth.go handlersStudent.go handlersBlog.go sessions.go" \