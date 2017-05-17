package cpu

import (
	"encoding/binary"
	"github.com/pkg/errors"
)

// wraps MemSim to make a Cpu interface-compatible memory model
type Mem struct {
	bits uint
	// methods return an error for addresses that do not fit inside mask
	// calculated by NewMem using ^uint64(0) >> (64 - bits)
	mask uint64
	// Mem.hooks is set when passing *Mem to NewHooks()
	hooks *Hooks
	// MemSim is private, so any cpu-facing functionality needs to be wrapped by Mem
	sim *MemSim

	order binary.ByteOrder
}

func NewMem(bits uint, order binary.ByteOrder) *Mem {
	return &Mem{
		bits:  bits,
		mask:  ^uint64(0) >> (64 - bits),
		sim:   &MemSim{},
		order: order,
	}
}

func (m *Mem) MemMapProt(addr, size uint64, prot int) error {
	if addr+size&m.mask != addr+size {
		return errors.New("region outside memory range")
	}
	m.sim.Map(addr, size, prot, false)
	return nil
}

func (m *Mem) MemProt(addr, size uint64, prot int) error {
	if mapped, _ := m.sim.RangeValid(addr, size, 0); !mapped {
		return errors.New("range not mapped")
	}
	m.sim.Prot(addr, size, prot)
	return nil
}

func (m *Mem) MemUnmap(addr, size uint64) error {
	if mapped, _ := m.sim.RangeValid(addr, size, 0); !mapped {
		return errors.New("range not mapped")
	}
	m.sim.Unmap(addr, size)
	return nil
}

func (m *Mem) MemReadInto(addr uint64, p []byte) error {
	return m.sim.Read(addr, p, 0)
}

func (m *Mem) MemRead(addr, size uint64) ([]byte, error) {
	p := make([]byte, size)
	if err := m.MemReadInto(addr, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (m *Mem) MemWrite(addr uint64, p []byte) error {
	return m.sim.Write(addr, p, 0)
}

// Read while checking protections. This exists to support a CPU interpreter.
func (m *Mem) ReadProt(addr, size uint64, prot int) ([]byte, error) {
	// TODO: issue read hook here?
	p := make([]byte, size)
	if err := m.sim.Read(addr, p, prot); err != nil {
		return nil, err
	}
	return p, nil
}

// Write while checking protections. This exists to support a CPU interpreter.
func (m *Mem) WriteProt(addr uint64, p []byte, prot int) error {
	// TODO: issue write hook here?
	return m.sim.Write(addr, p, prot)
}

func (m *Mem) ReadUint(addr uint64, size, prot int) (uint64, error) {
	if size > 8 {
		return 0, errors.Errorf("MemReadUint size too large: %d > 8", size)
	}
	p, err := m.ReadProt(addr, uint64(size), prot)
	if err != nil {
		return 0, err
	}
	return UnpackUint(m.order, size, p)
}

func (m *Mem) WriteUint(addr uint64, size, prot int, val uint64) error {
	var buf [8]byte
	if size > 8 {
		return errors.Errorf("MemWriteUint size too large: %d > 8", size)
	}
	if _, err := PackUint(m.order, size, buf[:], val); err != nil {
		return err
	}
	return m.WriteProt(addr, buf[:size], prot)
}
