package consensus

// GetConsensusPhase returns the current phase of the consensus
func (consensus *Consensus) GetConsensusPhase() string {
	return consensus.phase.String()
}

// GetConsensusMode returns the current mode of the consensus
func (consensus *Consensus) GetConsensusMode() string {
	return consensus.current.mode.String()
}

// GetCurViewID returns the current view ID of the consensus
func (consensus *Consensus) GetCurViewID() uint64 {
	return consensus.current.GetCurViewID()
}

// GetViewChangingID returns the current view changing ID of the consensus
func (consensus *Consensus) GetViewChangingID() uint64 {
	return consensus.current.GetViewChangingID()
}
