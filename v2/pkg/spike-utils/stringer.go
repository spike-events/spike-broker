package spike_utils

type Stringer string

func (s Stringer) String() string { return string(s) }
