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

func storagesInit(conf ConfOpts) (storagesMap map[string]interfaces.Storage, err error) {
	var (
		rl   int64
		errs *multierror.Error
	)

	storagesMap = make(map[string]interfaces.Storage)

	rl, err = getRateLimit(disk, nil, conf.Limits)
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("%s The limit won't be used for storage `local`", err))
	}
	storagesMap["local"] = local.Init(rl)

	for _, st := range conf.StorageConnects {
		if _, exist := storagesMap[st.Name]; exist {
			errs = multierror.Append(errs, fmt.Errorf("Storage with the name `%s` already defined. Please update configs ", st.Name))
			continue
		}

		if st.RateLimit != nil {
			rl, err = getRateLimit(net, &limitsConf{NetRate: st.RateLimit}, conf.Limits)
		} else {
			rl, err = getRateLimit(net, nil, conf.Limits)
		}
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("%s The limit won't be used for storage `%s`", err, st.Name))
		}

		switch {
		case st.S3Params != nil:
			storagesMap[st.Name], err = s3.Init(st.Name, s3.Opts(*st.S3Params), rl)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init storage `%s` with error: %w ", st.Name, err))
			}
		case st.ScpParams != nil:
			storagesMap[st.Name], err = sftp.Init(st.Name, sftp.Opts(*st.ScpParams), rl)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init storage `%s` with error: %w ", st.Name, err))
			}
		case st.SftpParams != nil:
			storagesMap[st.Name], err = sftp.Init(st.Name, sftp.Opts(*st.SftpParams), rl)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init storage `%s` with error: %w ", st.Name, err))
			}
		case st.FtpParams != nil:
			storagesMap[st.Name], err = ftp.Init(st.Name, ftp.Opts(*st.FtpParams), rl)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init storage `%s` with error: %w ", st.Name, err))
			}
		case st.NfsParams != nil:
			storagesMap[st.Name], err = nfs.Init(st.Name, nfs.Opts(*st.NfsParams), rl)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init storage `%s` with error: %w ", st.Name, err))
			}
		case st.WebDavParams != nil:
			storagesMap[st.Name], err = webdav.Init(st.Name, webdav.Opts(*st.WebDavParams), rl)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init storage `%s` with error: %w ", st.Name, err))
			}
		case st.SmbParams != nil:
			storagesMap[st.Name], err = smb.Init(st.Name, smb.Opts(*st.SmbParams), rl)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("Failed to init storage `%s` with error: %w ", st.Name, err))
			}
		default:
			errs = multierror.Append(errs, fmt.Errorf("unable to define `%s` storage connect type by its params. Allowed connect params: %s", st.Name, strings.Join(allowedConnectParams, ", ")))
		}
	}

	return storagesMap, errs.ErrorOrNil()
}
