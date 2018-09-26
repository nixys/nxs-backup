
# Source: https://github.com/bro/cmake/blob/master/InstallSymlink.cmake
macro(InstallSymlink _filepath _sympath)
	get_filename_component(_symname ${_sympath} NAME)
	get_filename_component(_installdir ${_sympath} PATH)

	install(CODE "execute_process(COMMAND ${CMAKE_COMMAND} -E create_symlink
			${_filepath}
			${CMAKE_CURRENT_BINARY_DIR}/${_symname})")
	install(FILES ${CMAKE_CURRENT_BINARY_DIR}/${_symname}
			DESTINATION ${_installdir})
endmacro(InstallSymlink)
