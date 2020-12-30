package downloader_leg

import (
	"context"
	"errors"
	"time"

	"github.com/harmony-one/harmony/core"
	"github.com/harmony-one/harmony/core/types"
	nodeconfig "github.com/harmony-one/harmony/internal/configs/node"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/harmony-one/harmony/p2p"
	"github.com/harmony-one/harmony/p2p/stream/sync"
	"github.com/rs/zerolog"
)

type (
	// Downloader is the structure used for downloading blocks through stream sync
	Downloader struct {
		syncProto *sync.Protocol
		bc        *core.BlockChain

		downloadC chan *downloadTask
		closeC    chan struct{}

		logger zerolog.Logger
	}

	downloadTask struct {
		errC chan error
	}
)

var (
	// ErrDownloadInProgress is the error happens when a new download task is triggered
	// but the there is already a downloading in progress
	ErrDownloadInProgress = errors.New("downloader already in progress")
)

// NewDownloader creates a new downloader
func NewDownloader(host p2p.Host, bc *core.BlockChain, network nodeconfig.NetworkType) *Downloader {
	syncProtocol := sync.NewProtocol(sync.Config{
		Chain:     bc,
		Host:      host.GetP2PHost(),
		Discovery: host.GetDiscovery(),
		ShardID:   nodeconfig.ShardID(bc.ShardID()),
		Network:   network,
	})
	host.AddStreamProtocol(syncProtocol)

	return &Downloader{
		syncProto: syncProtocol,
		bc:        bc,

		downloadC: make(chan *downloadTask),
		closeC:    make(chan struct{}),
		logger:    utils.Logger().With().Str("module", "downloader").Logger(),
	}
}

// Start start the downloader
func (d *Downloader) Start() {
	go d.loop()
	// bootstrap the downloader
	d.downloadC <- newDownloadTask()
}

// Close close the downloader
func (d *Downloader) Close() {
	close(d.closeC)
}

// DownloadAsync triggers the download async. If there is already a download task that is
// in progress, return ErrDownloadInProgress.
func (d *Downloader) DownloadAsync() <-chan error {
	task := newDownloadTask()
	select {
	case d.downloadC <- task:
	default:
		task.errC <- ErrDownloadInProgress
		close(task.errC)
	}
	return task.errC
}

func (d *Downloader) loop() {
	downloadCtx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(checkInterval)

	for {
		select {
		case <-ticker.C:
			d.downloadC <- newDownloadTask()

		case task := <-d.downloadC:
			err := d.doDownload(downloadCtx)
			if err != nil {
				d.logger.Warn().Err(err).Msg("failed to download")
			}
			task.errC <- err
			close(task.errC)

		case <-d.closeC:
			cancel()
			return
		}
	}
}

// doDownload is the main download process running in downloader module. Currently,
// a naive downloader protocol is running in single thread.
func (d *Downloader) doDownload(ctx context.Context) error {
	startHeight := d.bc.CurrentBlock().NumberU64()
	defer func() {
		endHeight := d.bc.CurrentBlock().NumberU64()
		d.logger.Info().Uint64("start height", startHeight).Uint64("end height", endHeight).
			Msg("doDownload finished")
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		bns := getBatchBlockNumbers(d.bc.CurrentBlock().NumberU64(), 1, downloadBatchSize)
		blocks, err := d.syncProto.GetBlocksByNumber(ctx, bns)
		if err != nil {
			return err
		}

		inserted, err := d.insertBlocks(blocks)
		if err != nil {
			return err
		}
		if inserted != len(blocks) {
			// nil block in blocks
			break
		}
	}
	return nil
}

func (d *Downloader) insertBlocks(blocks []*types.Block) (int, error) {
	for i, block := range blocks {
		if block == nil {
			blocks = blocks[:i]
			break
		}
	}
	return d.bc.InsertChain(blocks, true)
}

func newDownloadTask() *downloadTask {
	return &downloadTask{
		errC: make(chan error, 1),
	}
}

func getBatchBlockNumbers(start uint64, interval, size int) []uint64 {
	bns := make([]uint64, 0, size)
	cur := start
	for i := 0; i != size; i++ {
		bns = append(bns, cur)
		cur += uint64(interval)
	}
	return bns
}
