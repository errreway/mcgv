//go:build testing

package testing

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"regexp"
	"time"
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

func (suite *IgniteTestSuite) GridsCount() int {
	return len(suite.grids)
}

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

func (suite *IgniteTestSuite) WaitForTopologyVersion(topVer int, timeout time.Duration) bool {
	if len(suite.grids) == 0 {
		return false
	}
	reg, err := regexp.Compile(fmt.Sprintf("^Topology snapshot \\[ver=%d.*", topVer))
	require.NoError(suite.T(), err)
	return WaitForCondition(func() bool {
		for _, grid := range suite.grids {
			logFiles, err := GetLogFiles(grid.(*igniteInstanceImpl).params.InstanceIdx)
			require.NoError(suite.T(), err)
			res := false
			for _, logFile := range logFiles {
				res, err = MatchLog(reg, logFile)
				require.NoError(suite.T(), err)
				if !res {
					return false
				}
			}
		}
		return true
	}, timeout)
}
