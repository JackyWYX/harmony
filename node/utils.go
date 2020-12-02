package node

func (node *Node) StartDownloader() {
	if node.Downloader != nil {
		node.Downloader.Start()
	}
}

func (node *Node) StopDownloader() {
	if node.Downloader != nil {
		node.Downloader.Close()
	}
}
