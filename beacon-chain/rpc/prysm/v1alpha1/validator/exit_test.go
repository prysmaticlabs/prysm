package validator

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	opfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestRandom(t *testing.T) {
	junk := "134 555 98 1319 237 827 319 1101 1808 821 1788 1490 995 1289 227 1868 996 636 877 38 714 1863 653 1489 1988 1307 437 1910 1660 1030 523 1968 1717 1381 257 357 1466 1521 155 2 1950 1555 510 67 1959 924 1128 70 591 1053 1431 753 212 1476 436 1310 1837 683 99 393 1265 5 1292 487 159 464 1961 852 404 243 368 1124 1075 671 1093 1690 785 1585 1485 1084 1800 51 629 1631 1919 366 313 1028 1331 124 1953 999 1253 559 1887 1415 3 738 1237 132 181 680 1010 1397 1208 150 1257 17 1996 402 1185 757 898 1417 375 1487 232 1658 1274 967 1180 1915 834 1562 542 1554 312 1040 1716 1911 846 93 192 1338 1355 919 361 715 1134 1032 1967 531 1181 634 1546 439 396 1549 211 1760 24 1141 520 978 267 1118 587 1591 876 684 891 187 597 1801 585 841 1793"
	junk2 := "419 703 1123 1773 1081 1776 1488 780 589 1903 1341 1454 1626 448 671 1896 110 1976 1855 275 1001 283 1471 1617 493 1235 444 1906 1772 1924 1298 1040 773 888 584 520 1286 1482 1531 1619 774 1135 1340 13 1096 1291 1935 535 1716 1148 1121 631 1758 1240 32 391 550 758 1425 1004 1179 1484 1178 1024 735 1145 986 1417 143 1490 108 875 575 471 1948 1257 1878 1862 1106 700 720 992 1677 1239 842 867 1651 1146 706 1073 1622 843 783 816 1670 1988 1249 1373 461 1410 1978 1221 802 1068 1054 1371 155 456 1202 31 1051 898 1067 100 498 326 1621 1588 1887 1521 1996 1248 985 844 130 271 434 39 1163 1043 1569 1149 183 677 824 1962 1501 113 1154 1246 1863 374 874 625 179 1021 887 376 141 18 704 1066 1109 199 716 508 1649 1640 595 1652 886 1475 981 1979 624 1194 270"
	junk3 := "106 1570 1819 131 268 557 754 1597 29 1777 1064 325 1046 244 388 266 354 1897 1585 1147 49 1909 1690 1008 160 642 1055 1356 251 885 1524 1322 617 312 1207 1337 891 1358 24 1039 680 1119 1781 1561 1628 76 1045 1834 787 224 1473 1949 1762 502 662 1496 1444 712 650 567 1016 1616 1946 1139 495 134 1056 1381 1017 1228 302 1845 1983 2 1975 927 470 7 541 998 836 293 410 1180 1553 1107 450 1659 1186 644 1714 314 337 1967 859 1014 15 180 361 1090 1231 1105 196 10 1805 848 840 213 478 1084 1479 1951 682 1953 223 1769 1789 347 1159 922 1223 479 428 1261 139 768 932 1648 1700 152 1369 365 1166 1324 1366 987 1803 1441 661 1078 1082 1704 1981 1440 1111 239 19 873 881 300 1203 136 8 756 1669 1136 1408 771 920 1268 475 1880 855 396 747 1130"
	assert.DeepEqual(t, numSorter(junk, t), numSorter(junk2, t))
	assert.DeepEqual(t, numSorter(junk, t), numSorter(junk3, t))
}

func numSorter(obj string, t *testing.T) []int {
	nums := strings.Split(obj, " ")

	intNums := []int{}

	for _, n := range nums {
		nmbr, err := strconv.Atoi(n)
		assert.NoError(t, err)
		intNums = append(intNums, nmbr)
	}
	sort.Slice(intNums, func(i, j int) bool {
		return intNums[i] < intNums[j]
	})
	return intNums
}
func TestProposeExit_Notification(t *testing.T) {
	ctx := context.Background()

	deposits, keys, err := util.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	require.NoError(t, err)
	beaconState, err := transition.GenesisBeaconState(ctx, deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	require.NoError(t, err)
	epoch := types.Epoch(2048)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(epoch))))
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	// Set genesis time to be 100 epochs ago.
	offset := int64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	genesisTime := time.Now().Add(time.Duration(-100*offset) * time.Second)
	mockChainService := &mockChain.ChainService{State: beaconState, Root: genesisRoot[:], Genesis: genesisTime}
	server := &Server{
		HeadFetcher:       mockChainService,
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		TimeFetcher:       mockChainService,
		StateNotifier:     mockChainService.StateNotifier(),
		OperationNotifier: mockChainService.OperationNotifier(),
		ExitPool:          voluntaryexits.NewPool(),
		P2P:               mockp2p.NewTestP2P(t),
	}

	// Subscribe to operation notifications.
	opChannel := make(chan *feed.Event, 1024)
	opSub := server.OperationNotifier.OperationFeed().Subscribe(opChannel)
	defer opSub.Unsubscribe()

	// Send the request, expect a result on the state feed.
	validatorIndex := types.ValidatorIndex(0)
	req := &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			Epoch:          epoch,
			ValidatorIndex: validatorIndex,
		},
	}
	req.Signature, err = signing.ComputeDomainAndSign(beaconState, epoch, req.Exit, params.BeaconConfig().DomainVoluntaryExit, keys[0])
	require.NoError(t, err)

	resp, err := server.ProposeExit(context.Background(), req)
	require.NoError(t, err)
	expectedRoot, err := req.Exit.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedRoot[:], resp.ExitRoot)

	// Ensure the state notification was broadcast.
	notificationFound := false
	for !notificationFound {
		select {
		case event := <-opChannel:
			if event.Type == opfeed.ExitReceived {
				notificationFound = true
				data, ok := event.Data.(*opfeed.ExitReceivedData)
				assert.Equal(t, true, ok, "Entity is of the wrong type")
				assert.NotNil(t, data.Exit)
			}
		case <-opSub.Err():
			t.Error("Subscription to state notifier failed")
			return
		}
	}
}

func TestProposeExit_NoPanic(t *testing.T) {
	ctx := context.Background()

	deposits, keys, err := util.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	require.NoError(t, err)
	beaconState, err := transition.GenesisBeaconState(ctx, deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	require.NoError(t, err)
	epoch := types.Epoch(2048)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(epoch))))
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	// Set genesis time to be 100 epochs ago.
	offset := int64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	genesisTime := time.Now().Add(time.Duration(-100*offset) * time.Second)
	mockChainService := &mockChain.ChainService{State: beaconState, Root: genesisRoot[:], Genesis: genesisTime}
	server := &Server{
		HeadFetcher:       mockChainService,
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		TimeFetcher:       mockChainService,
		StateNotifier:     mockChainService.StateNotifier(),
		OperationNotifier: mockChainService.OperationNotifier(),
		ExitPool:          voluntaryexits.NewPool(),
		P2P:               mockp2p.NewTestP2P(t),
	}

	// Subscribe to operation notifications.
	opChannel := make(chan *feed.Event, 1024)
	opSub := server.OperationNotifier.OperationFeed().Subscribe(opChannel)
	defer opSub.Unsubscribe()

	req := &ethpb.SignedVoluntaryExit{}
	_, err = server.ProposeExit(context.Background(), req)
	require.ErrorContains(t, "voluntary exit does not exist", err, "Expected error for no exit existing")

	// Send the request, expect a result on the state feed.
	validatorIndex := types.ValidatorIndex(0)
	req = &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			Epoch:          epoch,
			ValidatorIndex: validatorIndex,
		},
	}

	_, err = server.ProposeExit(context.Background(), req)
	require.ErrorContains(t, "invalid signature provided", err, "Expected error for no signature exists")
	req.Signature = bytesutil.FromBytes48([fieldparams.BLSPubkeyLength]byte{})

	_, err = server.ProposeExit(context.Background(), req)
	require.ErrorContains(t, "invalid signature provided", err, "Expected error for invalid signature length")
	req.Signature, err = signing.ComputeDomainAndSign(beaconState, epoch, req.Exit, params.BeaconConfig().DomainVoluntaryExit, keys[0])
	require.NoError(t, err)
	resp, err := server.ProposeExit(context.Background(), req)
	require.NoError(t, err)
	expectedRoot, err := req.Exit.HashTreeRoot()
	require.NoError(t, err)
	assert.DeepEqual(t, expectedRoot[:], resp.ExitRoot)
}
