package system

import "syscall"

// fromStatT converts a syscall.Stat_t type to a system.Stat_t type
func fromStatT(s *syscall.Stat_t) (*StatT, error) {
	return &StatT{size: s.Size,
		mode:  uint32(s.Mode),
		uid:   s.Uid,
		gid:   s.Gid,
		rdev:  uint64(s.Rdev),
		dev:   uint64(s.Dev),
		ino:   s.Ino,
		nlink: uint64(s.Nlink),
		mtim:  s.Mtimespec}, nil
}
