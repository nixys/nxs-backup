if(RPM OR SRPM)

	if(EXISTS "${CMAKE_ROOT}/Modules/CPack.cmake")
		include(InstallRequiredSystemLibraries)

		set(CPACK_GENERATOR "RPM")

		#set(CPACK_RPM_PACKAGE_DEBUG "ON")

		# Set the common CentOS directories to exclude them from rpm package
		set(CPACK_RPM_EXCLUDE_FROM_AUTO_FILELIST_ADDITION
			"/etc/cron.d"
			"/usr/sbin"
			"/etc/cron.d"
			"/etc/logrotate.d"
			"/var"
			"/var/log"
			"/lib"
			"/lib/systemd"
			"/lib/systemd/system")

		if(SRPM)
			set(CPACK_RPM_PACKAGE_SOURCES "ON")
		endif(SRPM)

		execute_process(COMMAND lsb_release -sr COMMAND sed s/[.].*// OUTPUT_VARIABLE redhat_version_major OUTPUT_STRIP_TRAILING_WHITESPACE)
		execute_process(COMMAND uname -m OUTPUT_VARIABLE CPACK_RPM_PACKAGE_ARCHITECTURE OUTPUT_STRIP_TRAILING_WHITESPACE)

		set(PACKAGE_RELEASE "1")

		set(CPACK_PACKAGE_DESCRIPTION_FILE "${CMAKE_CURRENT_SOURCE_DIR}/build-scope/tpls/centos/description")
		set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "Summary")
		set(CPACK_PACKAGE_VENDOR "Nixys Ltd.")
		set(CPACK_PACKAGE_CONTACT "https://nixys.ru")
		set(CPACK_RPM_PACKAGE_LICENSE "GPLv3")
		set(CPACK_PACKAGE_VERSION_MAJOR "${MAJOR_VERSION}")
		set(CPACK_PACKAGE_VERSION_MINOR "${MINOR_VERSION}")
		set(CPACK_PACKAGE_VERSION_PATCH "${PATCH_VERSION}")
		set(CPACK_PACKAGE_FILE_NAME "${CMAKE_PROJECT_NAME}_${MAJOR_VERSION}.${MINOR_VERSION}.${CPACK_PACKAGE_VERSION_PATCH}-${PACKAGE_RELEASE}.el${redhat_version_major}.${CPACK_RPM_PACKAGE_ARCHITECTURE}")
		set(CPACK_SOURCE_PACKAGE_FILE_NAME "${CMAKE_PROJECT_NAME}_${MAJOR_VERSION}.${MINOR_VERSION}.${CPACK_PACKAGE_VERSION_PATCH}")
		set(CPACK_RPM_PACKAGE_REQUIRES "")
		set(CPACK_RPM_PACKAGE_CONFLICTS "nxs-backup < 2.0.0")
		set(CPACK_RPM_PACKAGE_RELEASE "${PACKAGE_RELEASE}%{?dist}")
		set(CPACK_PACKAGE_RELOCATABLE OFF)

		set(CPACK_RPM_PRE_INSTALL_SCRIPT_FILE "${CMAKE_CURRENT_SOURCE_DIR}/build-scope/tpls/centos/preinstall")
		set(CPACK_RPM_POST_INSTALL_SCRIPT_FILE "${CMAKE_CURRENT_SOURCE_DIR}/build-scope/tpls/centos/postinstall")
		set(CPACK_RPM_PRE_UNINSTALL_SCRIPT_FILE "${CMAKE_CURRENT_SOURCE_DIR}/build-scope/tpls/centos/preuninstall")
		set(CPACK_RPM_POST_UNINSTALL_SCRIPT_FILE "${CMAKE_CURRENT_SOURCE_DIR}/build-scope/tpls/centos/postuninstall")

		set(CPACK_COMPONENTS_ALL Libraries ApplicationData)
		include(CPack)
	endif(EXISTS "${CMAKE_ROOT}/Modules/CPack.cmake")
endif(RPM OR SRPM)
