package protocols

// Direct is the built-in node that sends traffic directly without a proxy.
type Direct struct{}

func (d *Direct) GetProtocolMode() Mode { return ModeDirect }
func (d *Direct) GetName() string       { return "direct" }
func (d *Direct) GetAddr() string       { return "" }
func (d *Direct) GetPort() int          { return 0 }
func (d *Direct) GetInfo() string       { return "Protocol: direct" }
func (d *Direct) GetLink() string       { return "direct://" }
