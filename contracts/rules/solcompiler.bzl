
_SOLIDITY_COMPILER_BINARY_BUILD = """
package(default_visibility = ["//visibility:public"])

exports_files(
  ['solc'],
  visibility = ['//visibility:public'],
)

exports_files(
  ['sol2go.sh'],
  visibility = ['//visibility:public'],
)

filegroup(
  name = 'file',
  srcs = ['solc', 'sol2go.sh'],
  visibility = ['//visibility:public'],
)"""

_SOLIDITY_SOL2GO = """
#!/usr/bin/env bash

ABIGEN="$2"
OUTFILE="$3"
PKG="$4"
SOLCOMPILER=$5
GO_ETHEREUM_IMPORTMAP=$6
GO_ETHEREUM_REPO_BASE=$7
GO_ETHEREUM_LABEL_NAME=$8

EXTERNAL_DIR={external_dir}
GO_ETHEREUM="$EXTERNAL_DIR/$GO_ETHEREUM_LABEL_NAME/"
OUTDIR=$(dirname $OUTFILE)

#Create src/github.com/ethereum/ tree needed by goimports
mkdir -p $OUTDIR/src/$GO_ETHEREUM_REPO_BASE
ln -f -s $GO_ETHEREUM  $OUTDIR/src/$GO_ETHEREUM_IMPORTMAP

#Generate abi file
$SOLCOMPILER --overwrite --bin --abi -o $OUTDIR $1
CONTRACTNAME=$($SOLCOMPILER  --abi  $1  | sed -n 's,^=.*:\(.*\) .*=$,\\1,p')

#Generate go files
ABIFILE=$OUTDIR/$CONTRACTNAME.abi
BINFILE=$OUTDIR/$CONTRACTNAME.bin
/usr/bin/env -i GOPATH=$OUTDIR/ $ABIGEN --bin $BINFILE --abi $ABIFILE --pkg $PKG -type $CONTRACTNAME > $3
"""

def _solc_fetch_impl(ctx):
  #Detect host (same code as go_download_sdk)
  if ctx.os.name == "linux":
    host = "linux_amd64"
    res = ctx.execute(["uname", "-p"])
    if res.return_code == 0:
      uname = res.stdout.strip()
      if uname == "s390x":
          host = "linux_s390x"
      elif uname == "ppc64le":
          host = "linux_ppc64le"
      elif uname == "i686":
          host = "linux_386"
    elif ctx.os.name == "mac os x":
        host = "darwin_amd64"
    elif ctx.os.name.startswith("windows"):
        host = "windows_amd64"
    elif ctx.os.name == "freebsd":
        host = "freebsd_amd64"
    else:
        fail("Unsupported operating system: " + ctx.os.name)

    #Get the right url according to host
    sdks = ctx.attr.sdks
    if host not in sdks:
      fail("Unsupported host {}".format(host))
    url, sha256 = ctx.attr.sdks[host]

    #Code similar to http_file but make the file executable
    ctx.download(url=url, output="solidity_compiler/solc", sha256=sha256, executable=True)
    ctx.file("WORKSPACE", "workspace(name = \"{name}\")".format(name = ctx.name))
    ctx.file("solidity_compiler/BUILD", _SOLIDITY_COMPILER_BINARY_BUILD)
    external_dir = ctx.path('..')
    ctx.file("solidity_compiler/sol2go.sh", _SOLIDITY_SOL2GO.format(external_dir=external_dir))

solc_fetch = repository_rule(
    _solc_fetch_impl,
    attrs = {
        "sdks": attr.string_list_dict()
    },
)
