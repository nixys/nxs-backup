package ctx

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/nixys/nxs-backup/interfaces"
	"github.com/nixys/nxs-backup/modules/storage/ftp"
	"github.com/nixys/nxs-backup/modules/storage/local"
	"github.com/nixys/nxs-backup/modules/storage/nfs"
	"github.com/nixys/nxs-backup/modules/storage/s3"
	"github.com/nixys/nxs-backup/modules/storage/sftp"
	"github.com/nixys/nxs-backup/modules/storage/smb"
	"github.com/nixys/nxs-backup/modules/storage/webdav"
)

var allowedConnectParams = []string{
	"s3_params",
	"scp_params",
	"sftp_params",
	"ftp_params",
	"smb_params",
	"nfs_params",
	"webdav_params",
}

func storagesInit(storageConnects []storageConnectConf, mainLim *limitsConf) (storagesMap map[string]interfaces.Storage, err error) {
	var (
		rl      int64
		errs    *multierror.Error
		storage interfaces.Storage
	)

	storagesMap = make(map[string]interfaces.Storage)

	rl, err = getRateLimit(mainLim.DiskRate)
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("%s The limit won't be used for storage `local`", err))
	}
	storagesMap["local"] = local.Init(rl)

	for _, st := range storageConnects {
		if _, exist := storagesMap[st.Name]; exist {
			errs = multierror.Append(errs, fmt.Errorf("Storage with the name `%s` already defined. Please update configs ", st.Name))
			continue
		}

		if st.RateLimit != nil {
			rl, err = getRateLimit(st.RateLimit)
		} else {
			rl, err = getRateLimit(mainLim.NetRate)
		}
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("%s The limit won't be used for storage `%s`", err, st.Name))
		}

		switch {
		case st.S3Params != nil:
			storage, err = s3.Init(st.Name, s3.Opts(*st.S3Params), rl)
		case st.ScpParams != nil:
			storage, err = sftp.Init(st.Name, sftp.Opts(*st.ScpParams), rl)
		case st.SftpParams != nil:
			storage, err = sftp.Init(st.Name, sftp.Opts(*st.SftpParams), rl)
		case st.FtpParams != nil:
			storage, err = ftp.Init(st.Name, ftp.Opts(*st.FtpParams), rl)
		case st.NfsParams != nil:
			storage, err = nfs.Init(st.Name, nfs.Opts(*st.NfsParams), rl)
		case st.WebDavParams != nil:
			storage, err = webdav.Init(st.Name, webdav.Opts(*st.WebDavParams), rl)
		case st.SmbParams != nil:
			storage, err = smb.Init(st.Name, smb.Opts(*st.SmbParams), rl)
		default:
			err = fmt.Errorf("unable to define `%s` storage connect type by its params. Allowed connect params: %s", st.Name, strings.Join(allowedConnectParams, ", "))
		}

		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("Failed to init storage `%s` with error: %w ", st.Name, err))
		} else {
			storagesMap[st.Name] = storage
		}
	}

	return storagesMap, errs.ErrorOrNil()
}
