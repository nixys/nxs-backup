set(PYTHON_VERSION_FILE_TPL "${CMAKE_CURRENT_SOURCE_DIR}/${PROJECT_SRC_DIR}/version.py.in")
set(PYTHON_VERSION_FILE "${CMAKE_CURRENT_SOURCE_DIR}/${PROJECT_SRC_DIR}/version.py")
set(PYTHON_SPEC_FILE "${CMAKE_MODULE_PATH}/app-python.spec")

# Specify Python packages to be downloaded and imported into your project
set(PYTHON_MODULES
"mysqlclient"
"pymongo"
"psutil"
"psycopg2-binary"
"redis"
"pyyaml"
"distro"
)

foreach(module ${PYTHON_MODULES})
	message(STATUS "Getting Python module: ${module}")
	execute_process (
		COMMAND bash -c "pip3 install ${module}"
		RESULT_VARIABLE res
		ERROR_VARIABLE err
	)
	if(NOT "${res}" STREQUAL "0")
		message(FATAL_ERROR "Python module install error: ${err}")
	endif()
endforeach(module)

if(EXISTS ${PYTHON_VERSION_FILE_TPL})
	configure_file("${PYTHON_VERSION_FILE_TPL}" "${PYTHON_VERSION_FILE}" @ONLY)
endif()

add_custom_target(${PROJECT_NAME}
	ALL
	COMMAND pyinstaller --distpath ${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_BIN_DIR} --workpath ${CMAKE_CURRENT_BINARY_DIR}/tmp --specpath ${CMAKE_CURRENT_BINARY_DIR}/tmp ${PYTHON_SPEC_FILE}
	WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR})

install(PROGRAMS ${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_BIN_DIR}/${PROJECT_NAME} DESTINATION sbin)
