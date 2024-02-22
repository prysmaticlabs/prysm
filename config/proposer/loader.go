package proposer

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v5/config"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	validatorService "github.com/prysmaticlabs/prysm/v5/config/validator/service"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/validator"
	validatorpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v5/validator/db/iface"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type SettingsType int

const (
	none SettingsType = iota
	defaultFlag
	fileFlag
	urlFlag
	onlyDB
)

type SettingsLoader struct {
	loadMethods []SettingsType
	existsInDB  bool
	db          iface.ValidatorDB
	options     *flagOptions
}

type flagOptions struct {
	builderConfig *validatorService.BuilderConfig
	gasLimit      validator.Uint64
	hasGasLimit   bool
}

// SettingsLoaderOption sets additional options that affect the proposer settings
type SettingsLoaderOption func(cliCtx *cli.Context, psl *SettingsLoader) error

// WithBuilderConfig applies the --enable-builder flag to proposer settings
func WithBuilderConfig() SettingsLoaderOption {
	return func(cliCtx *cli.Context, psl *SettingsLoader) error {
		if cliCtx.Bool(flags.EnableBuilderFlag.Name) {
			psl.options.builderConfig = &validatorService.BuilderConfig{
				Enabled:  true,
				GasLimit: validator.Uint64(params.BeaconConfig().DefaultBuilderGasLimit),
			}
		}
		return nil
	}
}

// WithGasLimit applies the --suggested-gas-limit flag to proposer settings
func WithGasLimit() SettingsLoaderOption {
	return func(cliCtx *cli.Context, psl *SettingsLoader) error {
		sgl := cliCtx.String(flags.BuilderGasLimitFlag.Name)
		if sgl != "" {
			gl, err := strconv.ParseUint(sgl, 10, 64)
			if err != nil {
				return errors.New("Gas Limit is not a uint64")
			}
			psl.options.gasLimit = reviewGasLimit(validator.Uint64(gl))
			psl.options.hasGasLimit = true
		}
		return nil
	}
}

// NewProposerSettingsLoader returns a new proposer settings loader that can process the proposer settings based on flag options
func NewProposerSettingsLoader(cliCtx *cli.Context, db iface.ValidatorDB, opts ...SettingsLoaderOption) (*SettingsLoader, error) {
	if cliCtx.IsSet(flags.ProposerSettingsFlag.Name) && cliCtx.IsSet(flags.ProposerSettingsURLFlag.Name) {
		return nil, fmt.Errorf("cannot specify both %s and %s flags; choose one method for specifying proposer settings", flags.ProposerSettingsFlag.Name, flags.ProposerSettingsURLFlag.Name)
	}
	psExists, err := db.ProposerSettingsExists(cliCtx.Context)
	if err != nil {
		return nil, err
	}
	psl := &SettingsLoader{db: db, options: &flagOptions{}}

	if cliCtx.IsSet(flags.SuggestedFeeRecipientFlag.Name) {
		psl.loadMethods = append(psl.loadMethods, defaultFlag)
	}
	if cliCtx.IsSet(flags.ProposerSettingsFlag.Name) {
		psl.loadMethods = append(psl.loadMethods, fileFlag)
	}
	if cliCtx.IsSet(flags.ProposerSettingsURLFlag.Name) {
		psl.loadMethods = append(psl.loadMethods, urlFlag)
	}
	if len(psl.loadMethods) == 0 {
		method := none
		if psExists {
			// override with db
			method = onlyDB
		}
		psl.loadMethods = append(psl.loadMethods, method)
	}

	if psExists {
		psl.existsInDB = true
	}
	for _, o := range opts {
		if err := o(cliCtx, psl); err != nil {
			return nil, err
		}
	}

	return psl, nil
}

// Load saves the proposer settings to the database
func (psl *SettingsLoader) Load(cliCtx *cli.Context) (*validatorService.ProposerSettings, error) {
	var fileConfig *validatorpb.ProposerSettingsPayload

	// override settings based on other options
	if psl.options.builderConfig != nil && psl.options.hasGasLimit {
		psl.options.builderConfig.GasLimit = psl.options.gasLimit
	}

	// check if database has settings already
	if psl.existsInDB {
		dbps, err := psl.db.ProposerSettings(cliCtx.Context)
		if err != nil {
			return nil, err
		}
		fileConfig = dbps.ToConsensus()
	}

	// start to process based on load method
	for _, method := range psl.loadMethods {
		switch method {
		case defaultFlag:
			suggestedFee := cliCtx.String(flags.SuggestedFeeRecipientFlag.Name)
			if !common.IsHexAddress(suggestedFee) {
				return nil, errors.New("default fee recipient is not a valid Ethereum address")
			}
			if err := config.WarnNonChecksummedAddress(suggestedFee); err != nil {
				return nil, err
			}
			defaultConfig := &validatorpb.ProposerOptionPayload{
				FeeRecipient: suggestedFee,
			}
			if psl.options.builderConfig != nil {
				defaultConfig.Builder = psl.options.builderConfig.ToConsensus()
			}
			if fileConfig == nil {
				fileConfig = &validatorpb.ProposerSettingsPayload{}
			}
			fileConfig.DefaultConfig = defaultConfig
		case fileFlag:
			var settingFromFile *validatorpb.ProposerSettingsPayload
			if err := config.UnmarshalFromFile(cliCtx.Context, cliCtx.String(flags.ProposerSettingsFlag.Name), &settingFromFile); err != nil {
				return nil, err
			}
			if settingFromFile == nil {
				return nil, errors.New("proposer settings is empty after unmarshalling from file")
			}
			fileConfig = psl.processProposerSettings(settingFromFile, fileConfig)
		case urlFlag:
			var settingFromURL *validatorpb.ProposerSettingsPayload
			if err := config.UnmarshalFromURL(cliCtx.Context, cliCtx.String(flags.ProposerSettingsURLFlag.Name), &settingFromURL); err != nil {
				return nil, err
			}
			if settingFromURL == nil {
				return nil, errors.New("proposer settings is empty after unmarshalling from url")
			}
			fileConfig = psl.processProposerSettings(settingFromURL, fileConfig)
		case onlyDB:
			fileConfig = psl.processProposerSettings(nil, fileConfig)
		case none:
			if psl.options.builderConfig != nil {
				// if there are no proposer settings provided, create a default where fee recipient is not populated, this will be skipped for validator registration on validators that don't have a fee recipient set.
				// skip saving to DB if only builder settings are provided until a trigger like keymanager API updates with fee recipient values
				option := &validatorService.ProposerOption{
					BuilderConfig: psl.options.builderConfig.Clone(),
				}
				if fileConfig == nil {
					fileConfig = &validatorpb.ProposerSettingsPayload{}
				}
				fileConfig.DefaultConfig = option.ToConsensus()
			}
		default:
			return nil, errors.New("load method for proposer settings does not exist")
		}
	}

	// exit early if nothing is provided
	if fileConfig == nil {
		log.Warn("No proposer settings were provided")
		return nil, nil
	}
	ps, err := validatorService.ProposerSettingFromConsensus(fileConfig)
	if err != nil {
		return nil, err
	}
	if err := psl.db.SaveProposerSettings(cliCtx.Context, ps); err != nil {
		return nil, err
	}
	return ps, nil
}

func (psl *SettingsLoader) processProposerSettings(loadedSettings, dbSettings *validatorpb.ProposerSettingsPayload) *validatorpb.ProposerSettingsPayload {
	dbOnly := false
	if loadedSettings == nil && dbSettings == nil {
		return nil
	}
	// fill in missing data from db
	if loadedSettings == nil && dbSettings != nil {
		dbOnly = true
		loadedSettings = dbSettings
	}
	if loadedSettings.DefaultConfig == nil && dbSettings != nil && dbSettings.DefaultConfig != nil {
		loadedSettings.DefaultConfig = dbSettings.DefaultConfig
	}
	if loadedSettings.ProposerConfig == nil && dbSettings != nil && dbSettings.ProposerConfig != nil {
		loadedSettings.ProposerConfig = dbSettings.ProposerConfig
	}
	// if default and proposer configs are both missing even after db setting
	if loadedSettings.DefaultConfig == nil && loadedSettings.ProposerConfig == nil {
		return nil
	}

	if loadedSettings.DefaultConfig != nil {
		if loadedSettings.DefaultConfig.Builder != nil {
			loadedSettings.DefaultConfig.Builder.GasLimit = reviewGasLimit(loadedSettings.DefaultConfig.Builder.GasLimit)
		}
		// override the db settings with the results based on whether the --enable-builder flag is provided.
		if psl.options.builderConfig == nil && dbOnly {
			loadedSettings.DefaultConfig.Builder = nil
		}
		if psl.options.builderConfig != nil {
			o := psl.options.builderConfig.ToConsensus()
			if loadedSettings.DefaultConfig.Builder != nil {
				// only override the enabled if builder settings exist
				loadedSettings.DefaultConfig.Builder.Enabled = o.Enabled
			} else {
				loadedSettings.DefaultConfig.Builder = o
			}
		} else if psl.options.hasGasLimit && loadedSettings.DefaultConfig.Builder != nil {
			loadedSettings.DefaultConfig.Builder.GasLimit = psl.options.gasLimit
		}
	}
	if loadedSettings.ProposerConfig != nil && len(loadedSettings.ProposerConfig) != 0 {
		for _, option := range loadedSettings.ProposerConfig {
			if option.Builder != nil {
				option.Builder.GasLimit = reviewGasLimit(option.Builder.GasLimit)
			}
			// override the db settings with the results based on whether the --enable-builder flag is provided.
			if psl.options.builderConfig == nil && dbOnly {
				option.Builder = nil
			}
			if psl.options.builderConfig != nil {
				o := psl.options.builderConfig.ToConsensus()
				if option.Builder != nil {
					// only override the enabled if builder settings exist
					option.Builder.Enabled = o.Enabled
				} else {
					option.Builder = o
				}
			} else if psl.options.hasGasLimit && option.Builder != nil {
				option.Builder.GasLimit = psl.options.gasLimit
			}
		}
	}

	return loadedSettings
}

func reviewGasLimit(gasLimit validator.Uint64) validator.Uint64 {
	// sets gas limit to default if not defined or set to 0
	if gasLimit == 0 {
		return validator.Uint64(params.BeaconConfig().DefaultBuilderGasLimit)
	}
	// TODO(10810): add in warning for ranges
	return gasLimit
}
