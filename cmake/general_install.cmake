# Place here the rules to install files, directories, etc for new packages

install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/build-scope/pkg/general/etc/cron.d/nxs-backup DESTINATION /etc/cron.d/)
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/build-scope/pkg/general/etc/logrotate.d/nxs-backup DESTINATION /etc/logrotate.d)

install(DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/build-scope/pkg/general/etc/nxs-backup DESTINATION /etc)
install(DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/build-scope/pkg/general/usr/share/nxs-backup DESTINATION /usr/share)
install(DIRECTORY DESTINATION /var/log/nxs-backup)
