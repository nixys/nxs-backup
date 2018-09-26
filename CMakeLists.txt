cmake_minimum_required(VERSION 2.8)

macro(GET_PROJECT_SETTING setting_name var)
	execute_process (
		COMMAND bash -c "nxs-build-tools --project-root=${CMAKE_CURRENT_SOURCE_DIR} --command=setting-get --setting=${setting_name}"
		OUTPUT_VARIABLE v
		RESULT_VARIABLE res
		ERROR_VARIABLE err
	)
	if(NOT "${res}" STREQUAL "0")
		message(FATAL_ERROR "Get project settings error (setting: ${setting_name}): ${err}")
	endif()
	set(${var} ${v})
endmacro(GET_PROJECT_SETTING)

GET_PROJECT_SETTING("project-name" PROJECT_NAME)
GET_PROJECT_SETTING("version-major" MAJOR_VERSION)
GET_PROJECT_SETTING("version-minor" MINOR_VERSION)
GET_PROJECT_SETTING("version-patch" PATCH_VERSION)

project(${PROJECT_NAME} NONE)

set(PROJECT_BIN_DIR "objs")
set(PROJECT_SRC_DIR "src") # For Go projects src dir can't be an absulote

set(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "${CMAKE_CURRENT_SOURCE_DIR}/cmake")

include(helpers_install)
include(general_install)

# Remove unnecessary modules
include(python)
include(rpm-build)
