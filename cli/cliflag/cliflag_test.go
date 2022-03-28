package cliflag_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/cli/cliflag"
	"github.com/coder/coder/cryptorand"
)

// Testcliflag cannot run in parallel because it uses t.Setenv.
//nolint:paralleltest
func TestCliflag(t *testing.T) {

	t.Run("StringDefault", func(t *testing.T) {
		var p string
		flagset, name, shorthand, env, usage := randomFlag()
		def, _ := cryptorand.String(10)

		cliflag.StringVarP(flagset, &p, name, shorthand, env, def, usage)
		got, err := flagset.GetString(name)
		require.NoError(t, err)
		require.Equal(t, def, got)
		require.Contains(t, flagset.FlagUsages(), usage)
		require.Contains(t, flagset.FlagUsages(), fmt.Sprintf("(uses $%s).", env))
	})

	t.Run("StringEnvVar", func(t *testing.T) {
		var p string
		flagset, name, shorthand, env, usage := randomFlag()
		envValue, _ := cryptorand.String(10)
		t.Setenv(env, envValue)
		def, _ := cryptorand.String(10)

		cliflag.StringVarP(flagset, &p, name, shorthand, env, def, usage)
		got, err := flagset.GetString(name)
		require.NoError(t, err)
		require.Equal(t, envValue, got)
	})

	t.Run("EmptyEnvVar", func(t *testing.T) {
		var p string
		flagset, name, shorthand, env, usage := randomFlag()
		env = ""
		def, _ := cryptorand.String(10)

		cliflag.StringVarP(flagset, &p, name, shorthand, env, def, usage)
		got, err := flagset.GetString(name)
		require.NoError(t, err)
		require.Equal(t, def, got)
		require.Contains(t, flagset.FlagUsages(), usage)
		require.NotContains(t, flagset.FlagUsages(), fmt.Sprintf("(uses $%s).", env))
	})

	t.Run("IntDefault", func(t *testing.T) {
		var p uint8
		flagset, name, shorthand, env, usage := randomFlag()
		def, _ := cryptorand.Int63n(10)

		cliflag.Uint8VarP(flagset, &p, name, shorthand, env, uint8(def), usage)
		got, err := flagset.GetUint8(name)
		require.NoError(t, err)
		require.Equal(t, uint8(def), got)
		require.Contains(t, flagset.FlagUsages(), usage)
		require.Contains(t, flagset.FlagUsages(), fmt.Sprintf("(uses $%s).", env))
	})

	t.Run("IntEnvVar", func(t *testing.T) {
		var p uint8
		flagset, name, shorthand, env, usage := randomFlag()
		envValue, _ := cryptorand.Int63n(10)
		t.Setenv(env, strconv.FormatUint(uint64(envValue), 10))
		def, _ := cryptorand.Int()

		cliflag.Uint8VarP(flagset, &p, name, shorthand, env, uint8(def), usage)
		got, err := flagset.GetUint8(name)
		require.NoError(t, err)
		require.Equal(t, uint8(envValue), got)
	})

	t.Run("IntFailParse", func(t *testing.T) {
		var p uint8
		flagset, name, shorthand, env, usage := randomFlag()
		envValue, _ := cryptorand.String(10)
		t.Setenv(env, envValue)
		def, _ := cryptorand.Int63n(10)

		cliflag.Uint8VarP(flagset, &p, name, shorthand, env, uint8(def), usage)
		got, err := flagset.GetUint8(name)
		require.NoError(t, err)
		require.Equal(t, uint8(def), got)
	})

	t.Run("BoolDefault", func(t *testing.T) {
		var p bool
		flagset, name, shorthand, env, usage := randomFlag()
		def, _ := cryptorand.Bool()

		cliflag.BoolVarP(flagset, &p, name, shorthand, env, def, usage)
		got, err := flagset.GetBool(name)
		require.NoError(t, err)
		require.Equal(t, def, got)
		require.Contains(t, flagset.FlagUsages(), usage)
		require.Contains(t, flagset.FlagUsages(), fmt.Sprintf("(uses $%s).", env))
	})

	t.Run("BoolEnvVar", func(t *testing.T) {
		var p bool
		flagset, name, shorthand, env, usage := randomFlag()
		envValue, _ := cryptorand.Bool()
		t.Setenv(env, strconv.FormatBool(envValue))
		def, _ := cryptorand.Bool()

		cliflag.BoolVarP(flagset, &p, name, shorthand, env, def, usage)
		got, err := flagset.GetBool(name)
		require.NoError(t, err)
		require.Equal(t, envValue, got)
	})

	t.Run("BoolFailParse", func(t *testing.T) {
		var p bool
		flagset, name, shorthand, env, usage := randomFlag()
		envValue, _ := cryptorand.String(10)
		t.Setenv(env, envValue)
		def, _ := cryptorand.Bool()

		cliflag.BoolVarP(flagset, &p, name, shorthand, env, def, usage)
		got, err := flagset.GetBool(name)
		require.NoError(t, err)
		require.Equal(t, def, got)
	})
}

func randomFlag() (*pflag.FlagSet, string, string, string, string) {
	fsname, _ := cryptorand.String(10)
	flagset := pflag.NewFlagSet(fsname, pflag.PanicOnError)
	name, _ := cryptorand.String(10)
	shorthand, _ := cryptorand.String(1)
	env, _ := cryptorand.String(10)
	usage, _ := cryptorand.String(10)

	return flagset, name, shorthand, env, usage
}
