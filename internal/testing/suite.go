//go:build testing

package testing

import (
	"fmt"
	"github.com/stretchr/testify/suite"
)

type IgniteTestSuite struct {
	suite.Suite
	grids []IgniteInstance
}

func (suite *IgniteTestSuite) KillAllGrids() {
	for _, ign := range suite.grids {
		_ = ign.Kill()
	}
	suite.grids = nil
}

func (suite *IgniteTestSuite) StartIgnite(opts ...func(params *IgniteParams)) (IgniteInstance, error) {
	if suite.grids == nil {
		suite.grids = make([]IgniteInstance, 0)
	}
	idx := len(suite.grids)
	opts = append(opts, WithInstanceIndex(idx))
	ign, err := StartIgnite(opts...)
	if err != nil {
		return nil, err
	}
	suite.grids = append(suite.grids, ign)
	return ign, nil
}

//func (suite *IgniteTestSuite) StartClient(opts ...func(options *ignite.ClientConfiguration) error) (ignite.Client, error) {
//	sink, _ := logger.NewStdErrSink(logger.DebugLevel) // level is ok, error ignored.
//	opts = append(opts, ignite.WithLoggingSink(sink))
//	return ignite.Start(opts...)
//}

func (suite *IgniteTestSuite) GetIgnite(idx int) IgniteInstance {
	if idx >= len(suite.grids) {
		return nil
	}
	return suite.grids[idx]
}

func (suite *IgniteTestSuite) KillIgnite(idx int) error {
	if idx >= len(suite.grids) {
		return fmt.Errorf("index %d exceeds size of started grids %d", idx, len(suite.grids))
	}
	ign := suite.grids[idx]
	defer func() {
		suite.grids = append(suite.grids[:idx], suite.grids[idx+1:]...)
	}()
	err := ign.Kill()
	return err
}
