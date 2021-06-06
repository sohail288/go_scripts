EXECUTABLE=tools
SRC_DIRECTORY=src
BUILD_DIRECTORY=dist
GO_SOURCES=$(shell find ${SRC_DIRECTORY} -name "*.go")
MAIN=main.go
DEPENDENCIES=${SRC_DIRECTORY}/${MAIN} $(filter-out ${SRC_DIRECTORY}/${MAIN}, ${GO_SOURCES})

# Scripts
SCRIPTS_DIRECTORY=cmd
SCRIPTS_DEST=${BUILD_DIRECTORY}/scripts
SCRIPTS_FILES=$(shell find ${SCRIPTS_DIRECTORY} -name "*.go")
SCRIPTS_EXECUTABLES=$(patsubst %, ${SCRIPTS_DEST}/%, $(basename $(notdir ${SCRIPTS_FILES})))


all: ${BUILD_DIRECTORY} ${BUILD_DIRECTORY}/${EXECUTABLE} ${SCRIPTS_DEST} ${SCRIPTS_EXECUTABLES}
	@echo "done"

run: all
	@${BUILD_DIRECTORY}/${EXECUTABLE}

# eg, make run-connect CLI_ARGS='https://google.com'
CLI_ARGS=
run-%: ${SCRIPTS_EXECUTABLES}
	./${SCRIPTS_DEST}/$* ${CLI_ARGS}

${BUILD_DIRECTORY}:
	mkdir -p ${BUILD_DIRECTORY}

${SCRIPTS_DEST}: ${BUILD_DIRECTORY}
	mkdir -p ${SCRIPTS_DEST}

${SCRIPTS_DEST}/%: ${SCRIPTS_DIRECTORY}/%
	# use stem patching with target matching
	go build -o $@ $</$*.go

${BUILD_DIRECTORY}/${EXECUTABLE}: ${MAIN} ${DEPENDENCIES}
	go build -o ${BUILD_DIRECTORY}/${EXECUTABLE} ${SRC_DIRECTORY}/${MAIN}

%.go: ;

.PHONY: clean debug

debug:
	@echo ${GO_SOURCES}
	@echo ${SCRIPTS_FILES}
	@echo ${SCRIPTS_EXECUTABLES}


clean:
	rm -rf ${BUILD_DIRECTORY}
