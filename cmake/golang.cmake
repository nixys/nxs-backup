set(GO_VERSION_FILE_TPL "${CMAKE_CURRENT_SOURCE_DIR}/${PROJECT_SRC_DIR}/version.go.in")
set(GO_VERSION_FILE "${CMAKE_CURRENT_SOURCE_DIR}/${PROJECT_SRC_DIR}/version.go")

# Specify Go packages to be downloaded and imported into your project
set(GO_PACKAGES
)

foreach(package ${GO_PACKAGES})
	message(STATUS "Getting Go package: ${package}")
	execute_process (
		COMMAND bash -c "go get ${package}"
		RESULT_VARIABLE res
		ERROR_VARIABLE err
	)
	if(NOT "${res}" STREQUAL "0")
		message(FATAL_ERROR "Go package download error: ${err}")
	endif()
endforeach(package)

if(EXISTS ${GO_VERSION_FILE_TPL})
	configure_file("${GO_VERSION_FILE_TPL}" "${GO_VERSION_FILE}" @ONLY)
endif()

add_custom_target(${PROJECT_NAME}
	ALL
	COMMAND go build -o ${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_BIN_DIR}/${PROJECT_NAME} ./${PROJECT_SRC_DIR}
	WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR})

install(PROGRAMS ${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_BIN_DIR}/${PROJECT_NAME} DESTINATION bin)
