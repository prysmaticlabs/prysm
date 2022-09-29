load("@prysm//tools/go:def.bzl", "go_repository", "maybe")  # gazelle:keep
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")  # gazelle:keep
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

# Prysm's third party / external dependencies.
#
##################################################################
#
#                    ██████████████████
#                  ██                  ██
#                ██  ██████████████████  ██
#              ██  ██████████████████████  ██
#            ██  ██████████████████████████  ██
#          ██  ██████████████████████████████  ██
#        ██  ██████████████████████████████████  ██
#      ██  ██████████████████████████████████████  ██
#      ██  ██████    ██      ████  ████    ██████  ██
#      ██  ████  ████████  ████  ██  ██  ██  ████  ██
#      ██  ████  ████████  ████  ██  ██  ██  ████  ██
#      ██  ██████  ██████  ████  ██  ██    ██████  ██
#      ██  ████████  ████  ████  ██  ██  ████████  ██
#      ██  ████████  ████  ████  ██  ██  ████████  ██
#      ██  ████    ██████  ██████  ████  ████████  ██
#      ██  ██████████████████████████████████████  ██
#        ██  ██████████████████████████████████  ██
#          ██  ██████████████████████████████  ██
#            ██  ██████████████████████████  ██
#              ██  ██████████████████████  ██
#                ██  ██████████████████  ██
#                  ██                  ██
#                    ██████████████████
#
##################################################################
#           Make sure you have read DEPENDENCIES.md!
##################################################################
def prysm_deps():
    go_repository(
        name = "co_honnef_go_tools",
        importpath = "honnef.co/go/tools",
        sum = "h1:qTakTkI6ni6LFD5sBwwsdSO+AQqbSIxOauHTTQKZ/7o=",
        version = "v0.1.3",
    )
    go_repository(
        name = "com_github_aead_siphash",
        importpath = "github.com/aead/siphash",
        sum = "h1:FwHfE/T45KPKYuuSAKyyvE+oPWcaQ+CUmFW0bPlM+kg=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_afex_hystrix_go",
        importpath = "github.com/afex/hystrix-go",
        sum = "h1:rFw4nCn9iMW+Vajsk51NtYIcwSTkXr+JGrMd36kTDJw=",
        version = "v0.0.0-20180502004556-fa1af6a1f4f5",
    )

    go_repository(
        name = "com_github_ajstarks_svgo",
        importpath = "github.com/ajstarks/svgo",
        sum = "h1:wVe6/Ea46ZMeNkQjjBW6xcqyQA/j5e0D6GytH95g0gQ=",
        version = "v0.0.0-20180226025133-644b8db467af",
    )

    go_repository(
        name = "com_github_alecthomas_template",
        importpath = "github.com/alecthomas/template",
        sum = "h1:JYp7IbQjafoB+tBA3gMyHYHrpOtNuDiK/uB5uXxq5wM=",
        version = "v0.0.0-20190718012654-fb15b899a751",
    )
    go_repository(
        name = "com_github_alecthomas_units",
        importpath = "github.com/alecthomas/units",
        sum = "h1:UQZhZ2O0vMHr2cI+DC1Mbh0TJxzA3RcLoMsFw+aXw7E=",
        version = "v0.0.0-20190924025748-f65c72e2690d",
    )
    go_repository(
        name = "com_github_allegro_bigcache",
        importpath = "github.com/allegro/bigcache",
        sum = "h1:hg1sY1raCwic3Vnsvje6TT7/pnZba83LeFck5NrFKSc=",
        version = "v1.2.1",
    )
    go_repository(
        name = "com_github_andreasbriese_bbloom",
        importpath = "github.com/AndreasBriese/bbloom",
        sum = "h1:cTp8I5+VIoKjsnZuH8vjyaysT/ses3EvZeaV/1UkF2M=",
        version = "v0.0.0-20190825152654-46b345b51c96",
    )

    go_repository(
        name = "com_github_andreyvit_diff",
        importpath = "github.com/andreyvit/diff",
        sum = "h1:bvNMNQO63//z+xNgfBlViaCIJKLlCJ6/fmUseuG0wVQ=",
        version = "v0.0.0-20170406064948-c7f18ee00883",
    )
    go_repository(
        name = "com_github_anmitsu_go_shlex",
        importpath = "github.com/anmitsu/go-shlex",
        sum = "h1:kFOfPq6dUM1hTo4JG6LR5AXSUEsOjtdm0kw0FtQtMJA=",
        version = "v0.0.0-20161002113705-648efa622239",
    )

    go_repository(
        name = "com_github_antihax_optional",
        importpath = "github.com/antihax/optional",
        sum = "h1:xK2lYat7ZLaVVcIuj82J8kIro4V6kDe0AUDFboUCwcg=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_apache_arrow_go_arrow",
        importpath = "github.com/apache/arrow/go/arrow",
        sum = "h1:nxAtV4VajJDhKysp2kdcJZsq8Ss1xSA0vZTkVHHJd0E=",
        version = "v0.0.0-20191024131854-af6fa24be0db",
    )
    go_repository(
        name = "com_github_apache_thrift",
        importpath = "github.com/apache/thrift",
        sum = "h1:5hryIiq9gtn+MiLVn0wP37kb/uTeRZgN08WoCsAhIhI=",
        version = "v0.13.0",
    )

    go_repository(
        name = "com_github_aristanetworks_fsnotify",
        importpath = "github.com/aristanetworks/fsnotify",
        sum = "h1:it2ydpY6k0aXB7qjb4vGhOYOL6YDC/sr8vhqwokFQwQ=",
        version = "v1.4.2",
    )
    go_repository(
        name = "com_github_aristanetworks_glog",
        importpath = "github.com/aristanetworks/glog",
        sum = "h1:Bmjk+DjIi3tTAU0wxGaFbfjGUqlxxSXARq9A96Kgoos=",
        version = "v0.0.0-20191112221043-67e8567f59f3",
    )

    go_repository(
        name = "com_github_aristanetworks_goarista",
        importpath = "github.com/aristanetworks/goarista",
        sum = "h1:XJH0YfVFKbq782tlNThzN/Ud5qm/cx6LXOA/P6RkTxc=",
        version = "v0.0.0-20200805130819-fd197cf57d96",
    )
    go_repository(
        name = "com_github_aristanetworks_splunk_hec_go",
        importpath = "github.com/aristanetworks/splunk-hec-go",
        sum = "h1:O7zlcm4ve7JvqTyEK3vSBh1LngLezraqcxv8Ya6tQFY=",
        version = "v0.3.3",
    )
    go_repository(
        name = "com_github_armon_circbuf",
        importpath = "github.com/armon/circbuf",
        sum = "h1:QEF07wC0T1rKkctt1RINW/+RMTVmiwxETico2l3gxJA=",
        version = "v0.0.0-20150827004946-bbbad097214e",
    )

    go_repository(
        name = "com_github_armon_consul_api",
        importpath = "github.com/armon/consul-api",
        sum = "h1:G1bPvciwNyF7IUmKXNt9Ak3m6u9DE1rF+RmtIkBpVdA=",
        version = "v0.0.0-20180202201655-eb2c6b5be1b6",
    )
    go_repository(
        name = "com_github_armon_go_metrics",
        importpath = "github.com/armon/go-metrics",
        sum = "h1:8GUt8eRujhVEGZFFEjBj46YV4rDjvGrNxb0KMWYkL2I=",
        version = "v0.0.0-20180917152333-f0300d1749da",
    )
    go_repository(
        name = "com_github_armon_go_radix",
        importpath = "github.com/armon/go-radix",
        sum = "h1:BUAU3CGlLvorLI26FmByPp2eC2qla6E1Tw+scpcg/to=",
        version = "v0.0.0-20180808171621-7fddfc383310",
    )
    go_repository(
        name = "com_github_aryann_difflib",
        importpath = "github.com/aryann/difflib",
        sum = "h1:pv34s756C4pEXnjgPfGYgdhg/ZdajGhyOvzx8k+23nw=",
        version = "v0.0.0-20170710044230-e206f873d14a",
    )

    go_repository(
        name = "com_github_aws_aws_lambda_go",
        importpath = "github.com/aws/aws-lambda-go",
        sum = "h1:SuCy7H3NLyp+1Mrfp+m80jcbi9KYWAs9/BXwppwRDzY=",
        version = "v1.13.3",
    )

    go_repository(
        name = "com_github_aws_aws_sdk_go",
        importpath = "github.com/aws/aws-sdk-go",
        sum = "h1:0xphMHGMLBrPMfxR2AmVjZKcMEESEgWF8Kru94BNByk=",
        version = "v1.27.0",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2",
        importpath = "github.com/aws/aws-sdk-go-v2",
        sum = "h1:BS+UYpbsElC82gB+2E2jiCBg36i8HlubTB/dO/moQ9c=",
        version = "v1.2.0",
    )

    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_config",
        importpath = "github.com/aws/aws-sdk-go-v2/config",
        sum = "h1:ZAoq32boMzcaTW9bcUacBswAmHTbvlvDJICgHFZuECo=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_credentials",
        importpath = "github.com/aws/aws-sdk-go-v2/credentials",
        sum = "h1:NbvWIM1Mx6sNPTxowHgS2ewXCRp+NGTzUYb/96FZJbY=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_feature_ec2_imds",
        importpath = "github.com/aws/aws-sdk-go-v2/feature/ec2/imds",
        sum = "h1:EtEU7WRaWliitZh2nmuxEXrN0Cb8EgPUFGIoTMeqbzI=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_internal_presigned_url",
        importpath = "github.com/aws/aws-sdk-go-v2/service/internal/presigned-url",
        sum = "h1:4AH9fFjUlVktQMznF+YN33aWNXaR4VgDXyP28qokJC0=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_route53",
        importpath = "github.com/aws/aws-sdk-go-v2/service/route53",
        sum = "h1:cKr6St+CtC3/dl/rEBJvlk7A/IN5D5F02GNkGzfbtVU=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_sso",
        importpath = "github.com/aws/aws-sdk-go-v2/service/sso",
        sum = "h1:37QubsarExl5ZuCBlnRP+7l1tNwZPBSTqpTBrPH98RU=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_sts",
        importpath = "github.com/aws/aws-sdk-go-v2/service/sts",
        sum = "h1:TJoIfnIFubCX0ACVeJ0w46HEH5MwjwYN4iFhuYIhfIY=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_aws_smithy_go",
        importpath = "github.com/aws/smithy-go",
        sum = "h1:D6CSsM3gdxaGaqXnPgOBCeL6Mophqzu7KJOu7zW78sU=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_azure_azure_pipeline_go",
        importpath = "github.com/Azure/azure-pipeline-go",
        sum = "h1:6oiIS9yaG6XCCzhgAgKFfIWyo4LLCiDhZot6ltoThhY=",
        version = "v0.2.2",
    )
    go_repository(
        name = "com_github_azure_azure_sdk_for_go_sdk_azcore",
        importpath = "github.com/Azure/azure-sdk-for-go/sdk/azcore",
        sum = "h1:qoVeMsc9/fh/yhxVaA0obYjVH/oI/ihrOoMwsLS9KSA=",
        version = "v0.21.1",
    )
    go_repository(
        name = "com_github_azure_azure_sdk_for_go_sdk_internal",
        importpath = "github.com/Azure/azure-sdk-for-go/sdk/internal",
        sum = "h1:E+m3SkZCN0Bf5q7YdTs5lSm2CYY3CK4spn5OmUIiQtk=",
        version = "v0.8.3",
    )
    go_repository(
        name = "com_github_azure_azure_sdk_for_go_sdk_storage_azblob",
        importpath = "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob",
        sum = "h1:Px2UA+2RvSSvv+RvJNuUB6n7rs5Wsel4dXLe90Um2n4=",
        version = "v0.3.0",
    )

    go_repository(
        name = "com_github_azure_azure_storage_blob_go",
        importpath = "github.com/Azure/azure-storage-blob-go",
        sum = "h1:MuueVOYkufCxJw5YZzF842DY2MBsp+hLuh2apKY0mck=",
        version = "v0.7.0",
    )

    go_repository(
        name = "com_github_azure_go_autorest_autorest",
        importpath = "github.com/Azure/go-autorest/autorest",
        sum = "h1:MRvx8gncNaXJqOoLmhNjUAKh33JJF8LyxPhomEtOsjs=",
        version = "v0.9.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_adal",
        importpath = "github.com/Azure/go-autorest/autorest/adal",
        sum = "h1:CxTzQrySOxDnKpLjFJeZAS5Qrv/qFPkgLjx5bOAi//I=",
        version = "v0.8.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_date",
        importpath = "github.com/Azure/go-autorest/autorest/date",
        sum = "h1:yW+Zlqf26583pE43KhfnhFcdmSWlm5Ew6bxipnr/tbM=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_autorest_mocks",
        importpath = "github.com/Azure/go-autorest/autorest/mocks",
        sum = "h1:qJumjCaCudz+OcqE9/XtEPfvtOjOmKaui4EOpFI6zZc=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_logger",
        importpath = "github.com/Azure/go-autorest/logger",
        sum = "h1:ruG4BSDXONFRrZZJ2GUXDiUyVpayPmb1GnWeHDdaNKY=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_azure_go_autorest_tracing",
        importpath = "github.com/Azure/go-autorest/tracing",
        sum = "h1:TRn4WjSnkcSy5AEG3pnbtFSwNtwzjr4VYyQflFE619k=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_github_bazelbuild_rules_go",
        importpath = "github.com/bazelbuild/rules_go",
        sum = "h1:Wxu7JjqnF78cKZbsBsARLSXx/jlGaSLCnUV3mTlyHvM=",
        version = "v0.23.2",
    )
    go_repository(
        name = "com_github_benbjohnson_clock",
        importpath = "github.com/benbjohnson/clock",
        sum = "h1:ip6w0uFQkncKQ979AypyG0ER7mqUSBdKLOgAle/AT8A=",
        version = "v1.3.0",
    )

    go_repository(
        name = "com_github_beorn7_perks",
        importpath = "github.com/beorn7/perks",
        sum = "h1:VlbKKnNfV8bJzeqoa4cOKqO6bYr3WgKZxO8Z16+hsOM=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_bgentry_go_netrc",
        importpath = "github.com/bgentry/go-netrc",
        sum = "h1:xDfNPAt8lFiC1UJrqV3uuy861HCTo708pDMbjHHdCas=",
        version = "v0.0.0-20140422174119-9fd32a8b3d3d",
    )

    go_repository(
        name = "com_github_bgentry_speakeasy",
        importpath = "github.com/bgentry/speakeasy",
        sum = "h1:ByYyxL9InA1OWqxJqqp2A5pYHUrCiAL6K3J+LKSsQkY=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_bketelsen_crypt",
        importpath = "github.com/bketelsen/crypt",
        sum = "h1:+0HFd5KSZ/mm3JmhmrDukiId5iR6w4+BdFtfSy4yWIc=",
        version = "v0.0.3-0.20200106085610-5cbc8cc4026c",
    )
    go_repository(
        name = "com_github_bmizerany_pat",
        importpath = "github.com/bmizerany/pat",
        sum = "h1:y4B3+GPxKlrigF1ha5FFErxK+sr6sWxQovRMzwMhejo=",
        version = "v0.0.0-20170815010413-6226ea591a40",
    )
    go_repository(
        name = "com_github_boltdb_bolt",
        importpath = "github.com/boltdb/bolt",
        sum = "h1:JQmyP4ZBrce+ZQu0dY660FMfatumYDLun9hBCUVIkF4=",
        version = "v1.3.1",
    )
    go_repository(
        name = "com_github_bradfitz_go_smtpd",
        importpath = "github.com/bradfitz/go-smtpd",
        sum = "h1:ckJgFhFWywOx+YLEMIJsTb+NV6NexWICk5+AMSuz3ss=",
        version = "v0.0.0-20170404230938-deb6d6237625",
    )

    go_repository(
        name = "com_github_bradfitz_gomemcache",
        importpath = "github.com/bradfitz/gomemcache",
        sum = "h1:7IjN4QP3c38xhg6wz8R3YjoU+6S9e7xBc0DAVLLIpHE=",
        version = "v0.0.0-20170208213004-1952afaa557d",
    )

    go_repository(
        name = "com_github_btcsuite_btcd",
        importpath = "github.com/btcsuite/btcd",
        sum = "h1:IB8cVQcC2X5mHbnfirLG5IZnkWYNTPlLZVrxUYSotbE=",
        version = "v0.23.1",
    )
    go_repository(
        name = "com_github_btcsuite_btcd_btcec_v2",
        importpath = "github.com/btcsuite/btcd/btcec/v2",
        sum = "h1:fzn1qaOt32TuLjFlkzYSsBC35Q3KUjT1SwPxiMSCF5k=",
        version = "v2.2.0",
    )
    go_repository(
        name = "com_github_btcsuite_btcd_btcutil",
        importpath = "github.com/btcsuite/btcd/btcutil",
        sum = "h1:MO4klnGY+EWJdoWF12Wkuf4AWDBPMpZNeN/jRLrklUU=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_btcsuite_btcd_chaincfg_chainhash",
        importpath = "github.com/btcsuite/btcd/chaincfg/chainhash",
        sum = "h1:q0rUy8C/TYNBQS1+CGKw68tLOFYSNEs0TFnxxnS9+4U=",
        version = "v1.0.1",
    )

    go_repository(
        name = "com_github_btcsuite_btclog",
        importpath = "github.com/btcsuite/btclog",
        sum = "h1:bAs4lUbRJpnnkd9VhRV3jjAVU7DJVjMaK+IsvSeZvFo=",
        version = "v0.0.0-20170628155309-84c8d2346e9f",
    )
    go_repository(
        name = "com_github_btcsuite_btcutil",
        importpath = "github.com/btcsuite/btcutil",
        sum = "h1:YtWJF7RHm2pYCvA5t0RPmAaLUhREsKuKd+SLhxFbFeQ=",
        version = "v1.0.3-0.20201208143702-a53e38424cce",
    )
    go_repository(
        name = "com_github_btcsuite_go_socks",
        importpath = "github.com/btcsuite/go-socks",
        sum = "h1:R/opQEbFEy9JGkIguV40SvRY1uliPX8ifOvi6ICsFCw=",
        version = "v0.0.0-20170105172521-4720035b7bfd",
    )
    go_repository(
        name = "com_github_btcsuite_goleveldb",
        importpath = "github.com/btcsuite/goleveldb",
        sum = "h1:Tvd0BfvqX9o823q1j2UZ/epQo09eJh6dTcRp79ilIN4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_btcsuite_snappy_go",
        importpath = "github.com/btcsuite/snappy-go",
        sum = "h1:ZxaA6lo2EpxGddsA8JwWOcxlzRybb444sgmeJQMJGQE=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_btcsuite_websocket",
        importpath = "github.com/btcsuite/websocket",
        sum = "h1:R8vQdOQdZ9Y3SkEwmHoWBmX1DNXhXZqlTpq6s4tyJGc=",
        version = "v0.0.0-20150119174127-31079b680792",
    )
    go_repository(
        name = "com_github_btcsuite_winsvc",
        importpath = "github.com/btcsuite/winsvc",
        sum = "h1:J9B4L7e3oqhXOcm+2IuNApwzQec85lE+QaikUcCs+dk=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_bufbuild_buf",
        importpath = "github.com/bufbuild/buf",
        sum = "h1:11zJVA0D4uJVGOC9h+oOVHrKKoBgMYIqJJ0d1Xt6oeQ=",
        version = "v0.37.0",
    )
    go_repository(
        name = "com_github_buger_jsonparser",
        importpath = "github.com/buger/jsonparser",
        sum = "h1:D21IyuvjDCshj1/qq+pCNd3VZOAEI9jy6Bi131YlXgI=",
        version = "v0.0.0-20181115193947-bf1c66bbce23",
    )

    go_repository(
        name = "com_github_burntsushi_toml",
        importpath = "github.com/BurntSushi/toml",
        sum = "h1:ksErzDEI1khOiGPgpwuI7x2ebx/uXQNw7xJpn9Eq1+I=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_burntsushi_xgb",
        importpath = "github.com/BurntSushi/xgb",
        sum = "h1:1BDTz0u9nC3//pOCMdNH+CiXJVYJh5UQNCOBG7jbELc=",
        version = "v0.0.0-20160522181843-27f122750802",
    )

    go_repository(
        name = "com_github_c_bata_go_prompt",
        importpath = "github.com/c-bata/go-prompt",
        sum = "h1:uyKRz6Z6DUyj49QVijyM339UJV9yhbr70gESwbNU3e0=",
        version = "v0.2.2",
    )
    go_repository(
        name = "com_github_casbin_casbin_v2",
        importpath = "github.com/casbin/casbin/v2",
        sum = "h1:bTwon/ECRx9dwBy2ewRVr5OiqjeXSGiTUY74sDPQi/g=",
        version = "v2.1.2",
    )
    go_repository(
        name = "com_github_cenkalti_backoff",
        importpath = "github.com/cenkalti/backoff",
        sum = "h1:tNowT99t7UNflLxfYYSlKYsBpXdEet03Pg2g16Swow4=",
        version = "v2.2.1+incompatible",
    )

    go_repository(
        name = "com_github_census_instrumentation_opencensus_proto",
        importpath = "github.com/census-instrumentation/opencensus-proto",
        sum = "h1:glEXhBS5PSLLv4IXzLA5yPRVX4bilULVyxxbrfOtDAk=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_cespare_cp",
        importpath = "github.com/cespare/cp",
        sum = "h1:nCb6ZLdB7NRaqsm91JtQTAme2SKJzXVsdPIPkyJr1MU=",
        version = "v1.1.1",
    )

    go_repository(
        name = "com_github_cespare_xxhash",
        importpath = "github.com/cespare/xxhash",
        sum = "h1:a6HrQnmkObjyL+Gs60czilIUGqrzKutQD6XZog3p+ko=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_cespare_xxhash_v2",
        importpath = "github.com/cespare/xxhash/v2",
        sum = "h1:YRXhKfTDauu4ajMg1TPgFO5jnlC2HCbmLXMcTG5cbYE=",
        version = "v2.1.2",
    )
    go_repository(
        name = "com_github_cheekybits_genny",
        importpath = "github.com/cheekybits/genny",
        sum = "h1:uGGa4nei+j20rOSeDeP5Of12XVm7TGUd4dJA9RDitfE=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_chzyer_logex",
        importpath = "github.com/chzyer/logex",
        sum = "h1:Swpa1K6QvQznwJRcfTfQJmTE72DqScAa40E+fbHEXEE=",
        version = "v1.1.10",
    )
    go_repository(
        name = "com_github_chzyer_readline",
        importpath = "github.com/chzyer/readline",
        sum = "h1:fY5BOSpyZCqRo5OhCuC+XN+r/bBCmeuuJtjz+bCNIf8=",
        version = "v0.0.0-20180603132655-2972be24d48e",
    )
    go_repository(
        name = "com_github_chzyer_test",
        importpath = "github.com/chzyer/test",
        sum = "h1:q763qf9huN11kDQavWsoZXJNW3xEE4JJyHa5Q25/sd8=",
        version = "v0.0.0-20180213035817-a1ea475d72b1",
    )
    go_repository(
        name = "com_github_cilium_ebpf",
        importpath = "github.com/cilium/ebpf",
        sum = "h1:QlHdikaxALkqWasW8hAC1mfR0jdmvbfaBdBPFmRSglA=",
        version = "v0.4.0",
    )

    go_repository(
        name = "com_github_clbanning_x2j",
        importpath = "github.com/clbanning/x2j",
        sum = "h1:EdRZT3IeKQmfCSrgo8SZ8V3MEnskuJP0wCYNpe+aiXo=",
        version = "v0.0.0-20191024224557-825249438eec",
    )

    go_repository(
        name = "com_github_client9_misspell",
        importpath = "github.com/client9/misspell",
        sum = "h1:ta993UF76GwbvJcIo3Y68y/M3WxlpEHPWIGDkJYwzJI=",
        version = "v0.3.4",
    )

    go_repository(
        name = "com_github_cloudflare_cloudflare_go",
        importpath = "github.com/cloudflare/cloudflare-go",
        sum = "h1:gFqGlGl/5f9UGXAaKapCGUfaTCgRKKnzu2VvzMZlOFA=",
        version = "v0.14.0",
    )
    go_repository(
        name = "com_github_cncf_udpa_go",
        importpath = "github.com/cncf/udpa/go",
        sum = "h1:cqQfy1jclcSy/FwLjemeg3SR1yaINm74aQyupQ0Bl8M=",
        version = "v0.0.0-20201120205902-5459f2c99403",
    )
    go_repository(
        name = "com_github_cncf_xds_go",
        importpath = "github.com/cncf/xds/go",
        sum = "h1:OZmjad4L3H8ncOIR8rnb5MREYqG8ixi5+WbeUsquF0c=",
        version = "v0.0.0-20210312221358-fbca930ec8ed",
    )

    go_repository(
        name = "com_github_cockroachdb_datadriven",
        importpath = "github.com/cockroachdb/datadriven",
        sum = "h1:OaNxuTZr7kxeODyLWsRMC+OD03aFUH+mW6r2d+MWa5Y=",
        version = "v0.0.0-20190809214429-80d97fb3cbaa",
    )
    go_repository(
        name = "com_github_codahale_hdrhistogram",
        importpath = "github.com/codahale/hdrhistogram",
        sum = "h1:qMd81Ts1T2OTKmB4acZcyKaMtRnY5Y44NuXGX2GFJ1w=",
        version = "v0.0.0-20161010025455-3a0bb77429bd",
    )

    go_repository(
        name = "com_github_consensys_bavard",
        importpath = "github.com/consensys/bavard",
        sum = "h1:+R8G1+Ftumd0DaveLgMIjrFPcAS4G8MsVXWXiyZL5BY=",
        version = "v0.1.8-0.20210406032232-f3452dc9b572",
    )
    go_repository(
        name = "com_github_consensys_gnark_crypto",
        importpath = "github.com/consensys/gnark-crypto",
        sum = "h1:C43yEtQ6NIf4ftFXD/V55gnGFgPbMQobd//YlnLjUJ8=",
        version = "v0.4.1-0.20210426202927-39ac3d4b3f1f",
    )
    go_repository(
        name = "com_github_consensys_goff",
        importpath = "github.com/consensys/goff",
        sum = "h1:eatQPk1I/aVec+F5qs47id17bWZsQFIjxu7C9MsrIHY=",
        version = "v0.3.10",
    )
    go_repository(
        name = "com_github_consensys_gurvy",
        importpath = "github.com/consensys/gurvy",
        sum = "h1:H2hvjvT2OFMgdMn5ZbhXqHt+F8DJ2clZW7Vmc0kFFxc=",
        version = "v0.3.8",
    )

    go_repository(
        name = "com_github_containerd_cgroups",
        build_file_proto_mode = "disable_global",
        importpath = "github.com/containerd/cgroups",
        sum = "h1:jN/mbWBEaz+T1pi5OFtnkQ+8qnmEbAr1Oo1FRm5B0dA=",
        version = "v1.0.4",
    )

    go_repository(
        name = "com_github_coreos_bbolt",
        importpath = "github.com/coreos/bbolt",
        sum = "h1:wZwiHHUieZCquLkDL0B8UhzreNWsPHooDAG3q34zk0s=",
        version = "v1.3.2",
    )
    go_repository(
        name = "com_github_coreos_etcd",
        importpath = "github.com/coreos/etcd",
        sum = "h1:8F3hqu9fGYLBifCmRCJsicFqDx/D68Rt3q1JMazcgBQ=",
        version = "v3.3.13+incompatible",
    )
    go_repository(
        name = "com_github_coreos_go_etcd",
        importpath = "github.com/coreos/go-etcd",
        sum = "h1:bXhRBIXoTm9BYHS3gE0TtQuyNZyeEMux2sDi4oo5YOo=",
        version = "v2.0.0+incompatible",
    )

    go_repository(
        name = "com_github_coreos_go_semver",
        importpath = "github.com/coreos/go-semver",
        sum = "h1:wkHLiw0WNATZnSG7epLsujiMCgPAc9xhjJ4tgnAxmfM=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_coreos_go_systemd",
        importpath = "github.com/coreos/go-systemd",
        sum = "h1:iW4rZ826su+pqaw19uhpSCzhj44qo35pNgKFGqzDKkU=",
        version = "v0.0.0-20191104093116-d3cd4ed1dbcf",
    )
    go_repository(
        name = "com_github_coreos_go_systemd_v22",
        importpath = "github.com/coreos/go-systemd/v22",
        sum = "h1:D9/bQk5vlXQFZ6Kwuu6zaiXJ9oTPe68++AzAJc1DzSI=",
        version = "v22.3.2",
    )

    go_repository(
        name = "com_github_coreos_pkg",
        importpath = "github.com/coreos/pkg",
        sum = "h1:lBNOc5arjvs8E5mO2tbpBpLoyyu8B6e44T7hJy6potg=",
        version = "v0.0.0-20180928190104-399ea9e2e55f",
    )

    go_repository(
        name = "com_github_cpuguy83_go_md2man",
        importpath = "github.com/cpuguy83/go-md2man",
        sum = "h1:BSKMNlYxDvnunlTymqtgONjNnaRV1sTpcovwwjF22jk=",
        version = "v1.0.10",
    )

    go_repository(
        name = "com_github_cpuguy83_go_md2man_v2",
        importpath = "github.com/cpuguy83/go-md2man/v2",
        sum = "h1:p1EgwI/C7NhT0JmVkwCD2ZBK8j4aeHQX2pMHHBfMQ6w=",
        version = "v2.0.2",
    )

    go_repository(
        name = "com_github_creack_pty",
        importpath = "github.com/creack/pty",
        sum = "h1:uDmaGzcdjhF4i/plgjmEsriH11Y0o7RKapEf/LDaM3w=",
        version = "v1.1.9",
    )
    go_repository(
        name = "com_github_cyberdelia_templates",
        importpath = "github.com/cyberdelia/templates",
        sum = "h1:/ovYnF02fwL0kvspmy9AuyKg1JhdTRUgPw4nUxd9oZM=",
        version = "v0.0.0-20141128023046-ca7fffd4298c",
    )

    go_repository(
        name = "com_github_d4l3k_messagediff",
        importpath = "github.com/d4l3k/messagediff",
        sum = "h1:ZcAIMYsUg0EAp9X+tt8/enBE/Q8Yd5kzPynLyKptt9U=",
        version = "v1.2.1",
    )

    go_repository(
        name = "com_github_data_dog_go_sqlmock",
        importpath = "github.com/DATA-DOG/go-sqlmock",
        sum = "h1:CWUqKXe0s8A2z6qCgkP4Kru7wC11YoAnoupUKFDnH08=",
        version = "v1.3.3",
    )
    go_repository(
        name = "com_github_dave_jennifer",
        importpath = "github.com/dave/jennifer",
        sum = "h1:S15ZkFMRoJ36mGAQgWL1tnr0NQJh9rZ8qatseX/VbBc=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_davidlazar_go_crypto",
        importpath = "github.com/davidlazar/go-crypto",
        sum = "h1:pFUpOrbxDR6AkioZ1ySsx5yxlDQZ8stG2b88gTPxgJU=",
        version = "v0.0.0-20200604182044-b73af7476f6c",
    )

    go_repository(
        name = "com_github_deckarep_golang_set",
        importpath = "github.com/deckarep/golang-set",
        sum = "h1:sk9/l/KqpunDwP7pSjUg0keiOOLEnOBHzykLrsPppp4=",
        version = "v1.8.0",
    )
    go_repository(
        name = "com_github_decred_dcrd_crypto_blake256",
        importpath = "github.com/decred/dcrd/crypto/blake256",
        sum = "h1:/8DMNYp9SGi5f0w7uCm6d6M4OU2rGFK09Y2A4Xv7EE0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_decred_dcrd_dcrec_secp256k1_v4",
        importpath = "github.com/decred/dcrd/dcrec/secp256k1/v4",
        sum = "h1:YLtO71vCjJRCBcrPMtQ9nqBsqpA1m5sE92cU+pd5Mcc=",
        version = "v4.0.1",
    )

    go_repository(
        name = "com_github_decred_dcrd_lru",
        importpath = "github.com/decred/dcrd/lru",
        sum = "h1:Kbsb1SFDsIlaupWPwsPp+dkxiBY1frcS07PCPgotKz8=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_deepmap_oapi_codegen",
        importpath = "github.com/deepmap/oapi-codegen",
        sum = "h1:SegyeYGcdi0jLLrpbCMoJxnUUn8GBXHsvr4rbzjuhfU=",
        version = "v1.8.2",
    )

    go_repository(
        name = "com_github_dgraph_io_badger",
        importpath = "github.com/dgraph-io/badger",
        sum = "h1:mNw0qs90GVgGGWylh0umH5iag1j6n/PeJtNvL6KY/x8=",
        version = "v1.6.2",
    )

    go_repository(
        name = "com_github_dgraph_io_ristretto",
        importpath = "github.com/dgraph-io/ristretto",
        sum = "h1:cNcG4c2n5xanQzp2hMyxDxPYVQmZ91y4WN6fJFlndLo=",
        version = "v0.0.4-0.20210318174700-74754f61e018",
    )
    go_repository(
        name = "com_github_dgrijalva_jwt_go",
        importpath = "github.com/dgrijalva/jwt-go",
        sum = "h1:7qlOGliEKZXTDg6OTjfoBKDXWrumCAMpl/TFQ4/5kLM=",
        version = "v3.2.0+incompatible",
    )

    go_repository(
        name = "com_github_dgryski_go_bitstream",
        importpath = "github.com/dgryski/go-bitstream",
        sum = "h1:akOQj8IVgoeFfBTzGOEQakCYshWD6RNo1M5pivFXt70=",
        version = "v0.0.0-20180413035011-3522498ce2c8",
    )
    go_repository(
        name = "com_github_dgryski_go_farm",
        importpath = "github.com/dgryski/go-farm",
        sum = "h1:tdlZCpZ/P9DhczCTSixgIKmwPv6+wP5DGjqLYw5SUiA=",
        version = "v0.0.0-20190423205320-6a90982ecee2",
    )
    go_repository(
        name = "com_github_dgryski_go_sip13",
        importpath = "github.com/dgryski/go-sip13",
        sum = "h1:RMLoZVzv4GliuWafOuPuQDKSm1SJph7uCRnnS61JAn4=",
        version = "v0.0.0-20181026042036-e10d5fee7954",
    )
    go_repository(
        name = "com_github_dlclark_regexp2",
        importpath = "github.com/dlclark/regexp2",
        sum = "h1:Izz0+t1Z5nI16/II7vuEo/nHjodOg0p7+OiDpjX5t1E=",
        version = "v1.4.1-0.20201116162257-a2a8dda75c91",
    )

    go_repository(
        name = "com_github_docker_docker",
        importpath = "github.com/docker/docker",
        sum = "h1:HlFGsy+9/xrgMmhmN+NGhCc5SHGJ7I+kHosRR1xc/aI=",
        version = "v1.6.2",
    )
    go_repository(
        name = "com_github_docker_go_units",
        importpath = "github.com/docker/go-units",
        sum = "h1:3uh0PgVws3nIA0Q+MwDC8yjEPf9zjRfZZWXZYDct3Tw=",
        version = "v0.4.0",
    )

    go_repository(
        name = "com_github_docker_spdystream",
        importpath = "github.com/docker/spdystream",
        sum = "h1:cenwrSVm+Z7QLSV/BsnenAOcDXdX4cMv4wP0B/5QbPg=",
        version = "v0.0.0-20160310174837-449fdfce4d96",
    )

    go_repository(
        name = "com_github_dop251_goja",
        importpath = "github.com/dop251/goja",
        sum = "h1:Yt+4K30SdjOkRoRRm3vYNQgR+/ZIy0RmeUDZo7Y8zeQ=",
        version = "v0.0.0-20220405120441-9037c2b61cbf",
    )
    go_repository(
        name = "com_github_dop251_goja_nodejs",
        importpath = "github.com/dop251/goja_nodejs",
        sum = "h1:tYwu/z8Y0NkkzGEh3z21mSWggMg4LwLRFucLS7TjARg=",
        version = "v0.0.0-20210225215109-d91c329300e7",
    )

    go_repository(
        name = "com_github_dustin_go_humanize",
        importpath = "github.com/dustin/go-humanize",
        sum = "h1:VSnTsYCnlFHaM2/igO1h6X3HA71jcobQuxemgkq4zYo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_dvyukov_go_fuzz",
        importpath = "github.com/dvyukov/go-fuzz",
        sum = "h1:NgO45/5mBLRVfiXerEFzH6ikcZ7DNRPS639xFg3ENzU=",
        version = "v0.0.0-20200318091601-be3528f3a813",
    )

    go_repository(
        name = "com_github_eapache_go_resiliency",
        importpath = "github.com/eapache/go-resiliency",
        sum = "h1:v7g92e/KSN71Rq7vSThKaWIq68fL4YHvWyiUKorFR1Q=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_eapache_go_xerial_snappy",
        importpath = "github.com/eapache/go-xerial-snappy",
        sum = "h1:YEetp8/yCZMuEPMUDHG0CW/brkkEp8mzqk2+ODEitlw=",
        version = "v0.0.0-20180814174437-776d5712da21",
    )
    go_repository(
        name = "com_github_eapache_queue",
        importpath = "github.com/eapache/queue",
        sum = "h1:YOEu7KNc61ntiQlcEeUIoDTJ2o8mQznoNvUhiigpIqc=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_eclipse_paho_mqtt_golang",
        importpath = "github.com/eclipse/paho.mqtt.golang",
        sum = "h1:1F8mhG9+aO5/xpdtFkW4SxOJB67ukuDC3t2y2qayIX0=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_edsrzf_mmap_go",
        importpath = "github.com/edsrzf/mmap-go",
        sum = "h1:CEBF7HpRnUCSJgGUb5h1Gm7e3VkmVDrR8lvWVLtrOFw=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_elastic_gosigar",
        importpath = "github.com/elastic/gosigar",
        sum = "h1:Dg80n8cr90OZ7x+bAax/QjoW/XqTI11RmA79ZwIm9/4=",
        version = "v0.14.2",
    )

    go_repository(
        name = "com_github_elazarl_goproxy",
        importpath = "github.com/elazarl/goproxy",
        sum = "h1:yUdfgN0XgIJw7foRItutHYUIhlcKzcSf5vDpdhQAKTc=",
        version = "v0.0.0-20180725130230-947c36da3153",
    )

    go_repository(
        name = "com_github_emicklei_dot",
        importpath = "github.com/emicklei/dot",
        sum = "h1:Ase39UD9T9fRBOb5ptgpixrxfx8abVzNWZi2+lr53PI=",
        version = "v0.11.0",
    )
    go_repository(
        name = "com_github_emicklei_go_restful",
        importpath = "github.com/emicklei/go-restful",
        sum = "h1:H2pdYOb3KQ1/YsqVWoWNLQO+fusocsw354rqGTZtAgw=",
        version = "v0.0.0-20170410110728-ff4f55a20633",
    )
    go_repository(
        name = "com_github_envoyproxy_go_control_plane",
        importpath = "github.com/envoyproxy/go-control-plane",
        sum = "h1:dulLQAYQFYtG5MTplgNGHWuV2D+OBD+Z8lmDBmbLg+s=",
        version = "v0.9.9-0.20210512163311-63b5d3c536b0",
    )
    go_repository(
        name = "com_github_envoyproxy_protoc_gen_validate",
        importpath = "github.com/envoyproxy/protoc-gen-validate",
        sum = "h1:EQciDnbrYxy13PgWoY8AqoxGiPrpgBZ1R8UNe3ddc+A=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_ethereum_go_ethereum",
        build_directives = [
            "gazelle:resolve go github.com/karalabe/usb @prysm//third_party/usb:go_default_library",
        ],
        importpath = "github.com/ethereum/go-ethereum",
        patch_args = ["-p1"],
        patches = [
            "//third_party:com_github_ethereum_go_ethereum_secp256k1.patch",
        ],
        sum = "h1:Xk8XAT4/UuqcjMLIMF+7imjkg32kfVFKoeyQDaO2yWM=",
        version = "v1.10.23",
    )

    go_repository(
        name = "com_github_evanphx_json_patch",
        importpath = "github.com/evanphx/json-patch",
        sum = "h1:fUDGZCv/7iAN7u0puUVhvKCcsR6vRfwrJatElLBEf0I=",
        version = "v4.2.0+incompatible",
    )

    go_repository(
        name = "com_github_fatih_color",
        importpath = "github.com/fatih/color",
        sum = "h1:8xPHl4/q1VyqGIPif1F+1V3Y3lSmrq01EabUW3CoW5s=",
        version = "v1.9.0",
    )
    go_repository(
        name = "com_github_ferranbt_fastssz",
        importpath = "github.com/ferranbt/fastssz",
        sum = "h1:9VDpsWq096+oGMDTT/SgBD/VgZYf4pTF+KTPmZ+OaKM=",
        version = "v0.0.0-20210120143747-11b9eff30ea9",
    )
    go_repository(
        name = "com_github_fjl_gencodec",
        importpath = "github.com/fjl/gencodec",
        sum = "h1:CndMRAH4JIwxbW8KYq6Q+cGWcGHz0FjGR3QqcInWcW0=",
        version = "v0.0.0-20220412091415-8bb9e558978c",
    )

    go_repository(
        name = "com_github_fjl_memsize",
        importpath = "github.com/fjl/memsize",
        sum = "h1:FtmdgXiUlNeRsoNMFlKLDt+S+6hbjVMEW6RGQ7aUf7c=",
        version = "v0.0.0-20190710130421-bcb5799ab5e5",
    )
    go_repository(
        name = "com_github_flynn_go_shlex",
        importpath = "github.com/flynn/go-shlex",
        sum = "h1:BHsljHzVlRcyQhjrss6TZTdY2VfCqZPbv5k3iBFa2ZQ=",
        version = "v0.0.0-20150515145356-3f9db97f8568",
    )

    go_repository(
        name = "com_github_flynn_noise",
        importpath = "github.com/flynn/noise",
        sum = "h1:DlTHqmzmvcEiKj+4RYo/imoswx/4r6iBlCMfVtrMXpQ=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_fogleman_gg",
        importpath = "github.com/fogleman/gg",
        sum = "h1:WXb3TSNmHp2vHoCroCIB1foO/yQ36swABL8aOVeDpgg=",
        version = "v1.2.1-0.20190220221249-0403632d5b90",
    )

    go_repository(
        name = "com_github_fortytw2_leaktest",
        importpath = "github.com/fortytw2/leaktest",
        sum = "h1:u8491cBMTQ8ft8aeV+adlcytMZylmA5nnwwkRZjI8vw=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_francoispqt_gojay",
        importpath = "github.com/francoispqt/gojay",
        sum = "h1:d2m3sFjloqoIUQU3TsHBgj6qg/BVGlTBeHDUmyJnXKk=",
        version = "v1.2.13",
    )

    go_repository(
        name = "com_github_franela_goblin",
        importpath = "github.com/franela/goblin",
        sum = "h1:gb2Z18BhTPJPpLQWj4T+rfKHYCHxRHCtRxhKKjRidVw=",
        version = "v0.0.0-20200105215937-c9ffbefa60db",
    )
    go_repository(
        name = "com_github_franela_goreq",
        importpath = "github.com/franela/goreq",
        sum = "h1:a9ENSRDFBUPkJ5lCgVZh26+ZbGyoVJG7yb5SSzF5H54=",
        version = "v0.0.0-20171204163338-bcd34c9993f8",
    )

    go_repository(
        name = "com_github_frankban_quicktest",
        importpath = "github.com/frankban/quicktest",
        sum = "h1:2QxQoC1TS09S7fhCPsrvqYdvP1H5M1P1ih5ABm3BTYk=",
        version = "v1.7.2",
    )
    go_repository(
        name = "com_github_fsnotify_fsnotify",
        importpath = "github.com/fsnotify/fsnotify",
        sum = "h1:jRbGcIw6P2Meqdwuo0H1p6JVLbL5DHKAKlYndzMwVZI=",
        version = "v1.5.4",
    )
    go_repository(
        name = "com_github_garslo_gogen",
        importpath = "github.com/garslo/gogen",
        sum = "h1:IZqZOB2fydHte3kUgxrzK5E1fW7RQGeDwE8F/ZZnUYc=",
        version = "v0.0.0-20170306192744-1d203ffc1f61",
    )

    go_repository(
        name = "com_github_garyburd_redigo",
        importpath = "github.com/garyburd/redigo",
        sum = "h1:0VruCpn7yAIIu7pWVClQC8wxCJEcG3nyzpMSHKi1PQc=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_gballet_go_libpcsclite",
        importpath = "github.com/gballet/go-libpcsclite",
        sum = "h1:f6D9Hr8xV8uYKlyuj8XIruxlh9WjVjdh1gIicAS7ays=",
        version = "v0.0.0-20191108122812-4678299bea08",
    )
    go_repository(
        name = "com_github_gdamore_encoding",
        importpath = "github.com/gdamore/encoding",
        sum = "h1:+7OoQ1Bc6eTm5niUzBa0Ctsh6JbMW6Ra+YNuAtDBdko=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_gdamore_tcell",
        importpath = "github.com/gdamore/tcell",
        sum = "h1:r35w0JBADPZCVQijYebl6YMWWtHRqVEGt7kL2eBADRM=",
        version = "v1.3.0",
    )

    go_repository(
        name = "com_github_getkin_kin_openapi",
        importpath = "github.com/getkin/kin-openapi",
        sum = "h1:6awGqF5nG5zkVpMsAih1QH4VgzS8phTxECUWIFo7zko=",
        version = "v0.61.0",
    )

    go_repository(
        name = "com_github_ghodss_yaml",
        importpath = "github.com/ghodss/yaml",
        sum = "h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_gliderlabs_ssh",
        importpath = "github.com/gliderlabs/ssh",
        sum = "h1:j3L6gSLQalDETeEg/Jg0mGY0/y/N6zI2xX1978P0Uqw=",
        version = "v0.1.1",
    )

    go_repository(
        name = "com_github_glycerine_go_unsnap_stream",
        importpath = "github.com/glycerine/go-unsnap-stream",
        sum = "h1:r04MMPyLHj/QwZuMJ5+7tJcBr1AQjpiAK/rZWRrQT7o=",
        version = "v0.0.0-20180323001048-9f0cb55181dd",
    )
    go_repository(
        name = "com_github_glycerine_goconvey",
        importpath = "github.com/glycerine/goconvey",
        sum = "h1:gclg6gY70GLy3PbkQ1AERPfmLMMagS60DKF78eWwLn8=",
        version = "v0.0.0-20190410193231-58a59202ab31",
    )

    go_repository(
        name = "com_github_go_chi_chi_v5",
        importpath = "github.com/go-chi/chi/v5",
        sum = "h1:DBPx88FjZJH3FsICfDAfIfnb7XxKIYVGG6lOPlhENAg=",
        version = "v5.0.0",
    )

    go_repository(
        name = "com_github_go_errors_errors",
        importpath = "github.com/go-errors/errors",
        sum = "h1:LUHzmkK3GUKUrL/1gfBUxAHzcev3apQlezX/+O7ma6w=",
        version = "v1.0.1",
    )

    go_repository(
        name = "com_github_go_gl_glfw",
        importpath = "github.com/go-gl/glfw",
        sum = "h1:QbL/5oDUmRBzO9/Z7Seo6zf912W/a6Sr4Eu0G/3Jho0=",
        version = "v0.0.0-20190409004039-e6da0acd62b1",
    )
    go_repository(
        name = "com_github_go_gl_glfw_v3_3_glfw",
        importpath = "github.com/go-gl/glfw/v3.3/glfw",
        sum = "h1:WtGNWLvXpe6ZudgnXrq0barxBImvnnJoMEhXAzcbM0I=",
        version = "v0.0.0-20200222043503-6f7a984d4dc4",
    )
    go_repository(
        name = "com_github_go_kit_kit",
        importpath = "github.com/go-kit/kit",
        sum = "h1:dXFJfIHVvUcpSgDOV+Ne6t7jXri8Tfv2uOLHUZ2XNuo=",
        version = "v0.10.0",
    )
    go_repository(
        name = "com_github_go_kit_log",
        importpath = "github.com/go-kit/log",
        sum = "h1:7i2K3eKTos3Vc0enKCfnVcgHh2olr/MyfboYq7cAcFw=",
        version = "v0.2.0",
    )

    go_repository(
        name = "com_github_go_logfmt_logfmt",
        importpath = "github.com/go-logfmt/logfmt",
        sum = "h1:otpy5pqBCBZ1ng9RQ0dPu4PN7ba75Y/aA+UpowDyNVA=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_go_logr_logr",
        importpath = "github.com/go-logr/logr",
        sum = "h1:fV3MLmabKIZ383XifUjFSwcoGee0v9qgPp8wy5svibE=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_go_ole_go_ole",
        importpath = "github.com/go-ole/go-ole",
        sum = "h1:t4MGB5xEDZvXI+0rMjjsfBsD7yAgp/s9ZDkL1JndXwY=",
        version = "v1.2.5",
    )
    go_repository(
        name = "com_github_go_openapi_jsonpointer",
        importpath = "github.com/go-openapi/jsonpointer",
        sum = "h1:gZr+CIYByUqjcgeLXnQu2gHYQC9o73G2XUeOFYEICuY=",
        version = "v0.19.5",
    )
    go_repository(
        name = "com_github_go_openapi_jsonreference",
        importpath = "github.com/go-openapi/jsonreference",
        sum = "h1:tF+augKRWlWx0J0B7ZyyKSiTyV6E1zZe+7b3qQlcEf8=",
        version = "v0.0.0-20160704190145-13c6e3589ad9",
    )
    go_repository(
        name = "com_github_go_openapi_spec",
        importpath = "github.com/go-openapi/spec",
        sum = "h1:C1JKChikHGpXwT5UQDFaryIpDtyyGL/CR6C2kB7F1oc=",
        version = "v0.0.0-20160808142527-6aced65f8501",
    )
    go_repository(
        name = "com_github_go_openapi_swag",
        importpath = "github.com/go-openapi/swag",
        sum = "h1:lTz6Ys4CmqqCQmZPBlbQENR1/GucA2bzYTE12Pw4tFY=",
        version = "v0.19.5",
    )
    go_repository(
        name = "com_github_go_playground_assert_v2",
        importpath = "github.com/go-playground/assert/v2",
        sum = "h1:MsBgLAaY856+nPRTKrp3/OZK38U/wa0CcBYNjji3q3A=",
        version = "v2.0.1",
    )
    go_repository(
        name = "com_github_go_playground_locales",
        importpath = "github.com/go-playground/locales",
        sum = "h1:u50s323jtVGugKlcYeyzC0etD1HifMjqmJqb8WugfUU=",
        version = "v0.14.0",
    )
    go_repository(
        name = "com_github_go_playground_universal_translator",
        importpath = "github.com/go-playground/universal-translator",
        sum = "h1:82dyy6p4OuJq4/CByFNOn/jYrnRPArHwAcmLoJZxyho=",
        version = "v0.18.0",
    )
    go_repository(
        name = "com_github_go_playground_validator_v10",
        importpath = "github.com/go-playground/validator/v10",
        sum = "h1:I7mrTYv78z8k8VXa/qJlOlEXn/nBh+BF8dHX5nt/dr0=",
        version = "v10.10.0",
    )

    go_repository(
        name = "com_github_go_sourcemap_sourcemap",
        importpath = "github.com/go-sourcemap/sourcemap",
        sum = "h1:W1iEw64niKVGogNgBN3ePyLFfuisuzeidWPMPWmECqU=",
        version = "v2.1.3+incompatible",
    )
    go_repository(
        name = "com_github_go_sql_driver_mysql",
        importpath = "github.com/go-sql-driver/mysql",
        sum = "h1:g24URVg0OFbNUTx9qqY1IRZ9D9z3iPyi5zKhQZpNwpA=",
        version = "v1.4.1",
    )

    go_repository(
        name = "com_github_go_stack_stack",
        importpath = "github.com/go-stack/stack",
        sum = "h1:5SgMzNM5HxrEjV0ww2lTmX6E2Izsfxas4+YHWRs3Lsk=",
        version = "v1.8.0",
    )
    go_repository(
        name = "com_github_go_task_slim_sprig",
        importpath = "github.com/go-task/slim-sprig",
        sum = "h1:p104kn46Q8WdvHunIJ9dAyjPVtrBPhSr3KT2yUst43I=",
        version = "v0.0.0-20210107165309-348f09dbbbc0",
    )

    go_repository(
        name = "com_github_go_yaml_yaml",
        importpath = "github.com/go-yaml/yaml",
        sum = "h1:RYi2hDdss1u4YE7GwixGzWwVo47T8UQwnTLB6vQiq+o=",
        version = "v2.1.0+incompatible",
    )
    go_repository(
        name = "com_github_godbus_dbus_v5",
        importpath = "github.com/godbus/dbus/v5",
        sum = "h1:4KLkAxT3aOY8Li4FRJe/KvhoNFFxo0m6fNuFUO8QJUk=",
        version = "v5.1.0",
    )

    go_repository(
        name = "com_github_gofrs_flock",
        importpath = "github.com/gofrs/flock",
        sum = "h1:MSdYClljsF3PbENUUEx85nkWfJSGfzYI9yEBZOJz6CY=",
        version = "v0.8.0",
    )
    go_repository(
        name = "com_github_gofrs_uuid",
        importpath = "github.com/gofrs/uuid",
        sum = "h1:1SD/1F5pU8p29ybwgQSwpQk+mwdRrXCYuPhW6m+TnJw=",
        version = "v4.0.0+incompatible",
    )

    go_repository(
        name = "com_github_gogo_googleapis",
        importpath = "github.com/gogo/googleapis",
        sum = "h1:kFkMAZBNAn4j7K0GiZr8cRYzejq68VbheufiV3YuyFI=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_gogo_protobuf",
        importpath = "github.com/gogo/protobuf",
        sum = "h1:Ov1cvc58UF3b5XjBnZv7+opcTcQFZebYjWzi34vdm4Q=",
        version = "v1.3.2",
    )

    go_repository(
        name = "com_github_golang_freetype",
        importpath = "github.com/golang/freetype",
        sum = "h1:DACJavvAHhabrF08vX0COfcOBJRhZ8lUbR+ZWIs0Y5g=",
        version = "v0.0.0-20170609003504-e2365dfdc4a0",
    )
    go_repository(
        name = "com_github_golang_gddo",
        importpath = "github.com/golang/gddo",
        sum = "h1:HoqgYR60VYu5+0BuG6pjeGp7LKEPZnHt+dUClx9PeIs=",
        version = "v0.0.0-20200528160355-8d077c1d8f4c",
    )

    go_repository(
        name = "com_github_golang_geo",
        importpath = "github.com/golang/geo",
        sum = "h1:lJwO/92dFXWeXOZdoGXgptLmNLwynMSHUmU6besqtiw=",
        version = "v0.0.0-20190916061304-5b978397cfec",
    )
    go_repository(
        name = "com_github_golang_glog",
        importpath = "github.com/golang/glog",
        sum = "h1:VKtxabqXZkF25pY9ekfRL6a582T4P37/31XEstQ5p58=",
        version = "v0.0.0-20160126235308-23def4e6c14b",
    )
    go_repository(
        name = "com_github_golang_groupcache",
        importpath = "github.com/golang/groupcache",
        sum = "h1:1r7pUrabqp18hOBcwBwiTsbnFeTZHV9eER/QT5JVZxY=",
        version = "v0.0.0-20200121045136-8c9f03a8e57e",
    )
    go_repository(
        name = "com_github_golang_jwt_jwt_v4",
        importpath = "github.com/golang-jwt/jwt/v4",
        sum = "h1:kHL1vqdqWNfATmA0FNMdmZNMyZI1U6O31X4rlIPoBog=",
        version = "v4.3.0",
    )
    go_repository(
        name = "com_github_golang_lint",
        importpath = "github.com/golang/lint",
        sum = "h1:2hRPrmiwPrp3fQX967rNJIhQPtiGXdlQWAxKbKw3VHA=",
        version = "v0.0.0-20180702182130-06c8688daad7",
    )

    go_repository(
        name = "com_github_golang_mock",
        importpath = "github.com/golang/mock",
        sum = "h1:ErTB+efbowRARo13NNdxyJji2egdxLGQhRaY+DUumQc=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_golang_protobuf",
        importpath = "github.com/golang/protobuf",
        patch_args = ["-p1"],
        patches = ["@io_bazel_rules_go//third_party:com_github_golang_protobuf-extras.patch"],
        sum = "h1:ROPKBNFfQgOUMifHyP+KYbvpjbdoFNs+aK7DXlji0Tw=",
        version = "v1.5.2",
    )

    go_repository(
        name = "com_github_golang_snappy",
        importpath = "github.com/golang/snappy",
        sum = "h1:yAGX7huGHXlcLOEtBnF4w7FQwA26wojNCwOYAEhLjQM=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_golangci_lint_1",
        importpath = "github.com/golangci/lint-1",
        sum = "h1:utua3L2IbQJmauC5IXdEA547bcoU5dozgQAfc8Onsg4=",
        version = "v0.0.0-20181222135242-d2cdd8c08219",
    )

    go_repository(
        name = "com_github_google_btree",
        importpath = "github.com/google/btree",
        sum = "h1:0udJVsspx3VBr5FwtLhQQtuAsVc79tTq0ocGIPAU6qo=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_google_flatbuffers",
        importpath = "github.com/google/flatbuffers",
        sum = "h1:O7CEyB8Cb3/DmtxODGtLHcEvpr81Jm5qLg/hsHnxA2A=",
        version = "v1.11.0",
    )
    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        sum = "h1:Khx7svrCpmxxtHBq5j2mp/xVjsi8hQMfNLvJFAlrGgU=",
        version = "v0.5.5",
    )
    go_repository(
        name = "com_github_google_go_github",
        importpath = "github.com/google/go-github",
        sum = "h1:N0LgJ1j65A7kfXrZnUDaYCs/Sf4rEjNlfyDHW9dolSY=",
        version = "v17.0.0+incompatible",
    )
    go_repository(
        name = "com_github_google_go_querystring",
        importpath = "github.com/google/go-querystring",
        sum = "h1:Xkwi/a1rcvNg1PPYe5vI8GbeBY/jrVuDX5ASuANWTrk=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_google_gofuzz",
        importpath = "github.com/google/gofuzz",
        sum = "h1:xRy4A+RhZaiKjJ1bPfwQ8sedCA+YS2YcCHW6ec7JMi0=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_google_gopacket",
        importpath = "github.com/google/gopacket",
        sum = "h1:ves8RnFZPGiFnTS0uPQStjwru6uO6h+nlr9j6fL7kF8=",
        version = "v1.1.19",
    )
    go_repository(
        name = "com_github_google_martian",
        importpath = "github.com/google/martian",
        sum = "h1:/CP5g8u/VJHijgedC/Legn3BAbAaWPgecwXBIDzw5no=",
        version = "v2.1.0+incompatible",
    )

    go_repository(
        name = "com_github_google_martian_v3",
        importpath = "github.com/google/martian/v3",
        sum = "h1:pMen7vLs8nvgEYhywH3KDWJIJTeEr2ULsVWHWYHQyBs=",
        version = "v3.0.0",
    )
    go_repository(
        name = "com_github_google_pprof",
        importpath = "github.com/google/pprof",
        sum = "h1:yAJXTCF9TqKcTiHJAE8dj7HMvPfh66eeA2JYW7eFpSE=",
        version = "v0.0.0-20210407192527-94a9f03dee38",
    )
    go_repository(
        name = "com_github_google_renameio",
        importpath = "github.com/google/renameio",
        sum = "h1:GOZbcHa3HfsPKPlmyPyN2KEohoMXOhdMbHrvbpl2QaA=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_google_uuid",
        importpath = "github.com/google/uuid",
        sum = "h1:t6JiXgmwXMjEs8VusXIJk2BXHsn+wx8BZdTaoZ5fu7I=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_googleapis_gax_go",
        importpath = "github.com/googleapis/gax-go",
        sum = "h1:j0GKcs05QVmm7yesiZq2+9cxHkNK9YM6zKx4D2qucQU=",
        version = "v2.0.0+incompatible",
    )

    go_repository(
        name = "com_github_googleapis_gax_go_v2",
        importpath = "github.com/googleapis/gax-go/v2",
        sum = "h1:sjZBwGj9Jlw33ImPtvFviGYvseOtDM7hkSKB7+Tv3SM=",
        version = "v2.0.5",
    )
    go_repository(
        name = "com_github_googleapis_gnostic",
        build_directives = [
            "gazelle:resolve go github.com/googleapis/gnostic/extensions //extensions:go_default_library",
        ],
        build_naming_convention = "go_default_library",
        importpath = "github.com/googleapis/gnostic",
        sum = "h1:rVsPeBmXbYv4If/cumu1AzZPwV58q433hvONV1UEZoI=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_gophercloud_gophercloud",
        importpath = "github.com/gophercloud/gophercloud",
        sum = "h1:P/nh25+rzXouhytV2pUHBb65fnds26Ghl8/391+sT5o=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_gopherjs_gopherjs",
        importpath = "github.com/gopherjs/gopherjs",
        sum = "h1:EGx4pi6eqNxGaHF6qqu48+N2wcFQ5qg5FXgOdqsJ5d8=",
        version = "v0.0.0-20181017120253-0766667cb4d1",
    )
    go_repository(
        name = "com_github_gordonklaus_ineffassign",
        importpath = "github.com/gordonklaus/ineffassign",
        sum = "h1:vc7Dmrk4JwS0ZPS6WZvWlwDflgDTA26jItmbSj83nug=",
        version = "v0.0.0-20200309095847-7953dde2c7bf",
    )
    go_repository(
        name = "com_github_gorilla_context",
        importpath = "github.com/gorilla/context",
        sum = "h1:AWwleXJkX/nhcU9bZSnZoi3h/qGYqQAGhq6zZe/aQW8=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_gorilla_mux",
        importpath = "github.com/gorilla/mux",
        sum = "h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=",
        version = "v1.8.0",
    )

    go_repository(
        name = "com_github_gorilla_websocket",
        importpath = "github.com/gorilla/websocket",
        sum = "h1:PPwGk2jz7EePpoHN/+ClbZu8SPxiqlu12wZP/3sWmnc=",
        version = "v1.5.0",
    )
    go_repository(
        name = "com_github_gostaticanalysis_comment",
        importpath = "github.com/gostaticanalysis/comment",
        sum = "h1:hlnx5+S2fY9Zo9ePo4AhgYsYHbM2+eAv8m/s1JiCd6Q=",
        version = "v1.4.2",
    )
    go_repository(
        name = "com_github_gostaticanalysis_testutil",
        importpath = "github.com/gostaticanalysis/testutil",
        sum = "h1:d2/eIbH9XjD1fFwD5SHv8x168fjbQ9PB8hvs8DSEC08=",
        version = "v0.3.1-0.20210208050101-bfb5c8eec0e4",
    )

    go_repository(
        name = "com_github_graph_gophers_graphql_go",
        importpath = "github.com/graph-gophers/graphql-go",
        sum = "h1:Eb9x/q6MFpCLz7jBCiP/WTxjSDrYLR1QY41SORZyNJ0=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_gregjones_httpcache",
        importpath = "github.com/gregjones/httpcache",
        sum = "h1:pdN6V1QBWetyv/0+wjACpqVH+eVULgEjkurDLq3goeM=",
        version = "v0.0.0-20180305231024-9cad4c3443a7",
    )

    go_repository(
        name = "com_github_grpc_ecosystem_go_grpc_middleware",
        importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
        sum = "h1:FlFbCRLd5Jr4iYXZufAvgWN6Ao0JrI5chLINnUXDDr0=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_grpc_ecosystem_go_grpc_prometheus",
        importpath = "github.com/grpc-ecosystem/go-grpc-prometheus",
        sum = "h1:Ovs26xHkKqVztRpIrF/92BcuyuQ/YW4NSIpoGtfXNho=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_grpc_ecosystem_grpc_gateway",
        importpath = "github.com/grpc-ecosystem/grpc-gateway",
        sum = "h1:gmcG1KaJ57LophUzW0Hy8NmPhnMZb4M0+kPpLofRdBo=",
        version = "v1.16.0",
    )
    go_repository(
        name = "com_github_grpc_ecosystem_grpc_gateway_v2",
        importpath = "github.com/grpc-ecosystem/grpc-gateway/v2",
        replace = "github.com/prysmaticlabs/grpc-gateway/v2",
        sum = "h1:elSfkWAcpL94zg7B9J2ozu+WcbMxD1rZ3rzUjYAy6R8=",
        version = "v2.3.1-0.20220721162526-0d1c40b5f064",
    )
    go_repository(
        name = "com_github_gxed_hashland_keccakpg",
        importpath = "github.com/gxed/hashland/keccakpg",
        sum = "h1:wrk3uMNaMxbXiHibbPO4S0ymqJMm41WiudyFSs7UnsU=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_gxed_hashland_murmur3",
        importpath = "github.com/gxed/hashland/murmur3",
        sum = "h1:SheiaIt0sda5K+8FLz952/1iWS9zrnKsEJaOJu4ZbSc=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_consul_api",
        importpath = "github.com/hashicorp/consul/api",
        sum = "h1:HXNYlRkkM/t+Y/Yhxtwcy02dlYwIaoxzvxPnS+cqy78=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_hashicorp_consul_sdk",
        importpath = "github.com/hashicorp/consul/sdk",
        sum = "h1:UOxjlb4xVNF93jak1mzzoBatyFju9nrkxpVwIp/QqxQ=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_hashicorp_errwrap",
        importpath = "github.com/hashicorp/errwrap",
        sum = "h1:hLrqtEDnRye3+sgx6z4qVLNuviH3MR5aQ0ykNJa/UYA=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_bexpr",
        importpath = "github.com/hashicorp/go-bexpr",
        sum = "h1:9kuI5PFotCboP3dkDYFr/wi0gg0QVbSNz5oFRpxn4uE=",
        version = "v0.1.10",
    )

    go_repository(
        name = "com_github_hashicorp_go_cleanhttp",
        importpath = "github.com/hashicorp/go-cleanhttp",
        sum = "h1:dH3aiDG9Jvb5r5+bYHsikaOUIpcM0xvgMXVoDkXMzJM=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_hashicorp_go_immutable_radix",
        importpath = "github.com/hashicorp/go-immutable-radix",
        sum = "h1:AKDB1HM5PWEA7i4nhcpwOrO2byshxBjXVn/J/3+z5/0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_msgpack",
        importpath = "github.com/hashicorp/go-msgpack",
        sum = "h1:zKjpN5BK/P5lMYrLmBHdBULWbJ0XpYR+7NGzqkZzoD4=",
        version = "v0.5.3",
    )
    go_repository(
        name = "com_github_hashicorp_go_multierror",
        importpath = "github.com/hashicorp/go-multierror",
        sum = "h1:iVjPR7a6H0tWELX5NxNe7bYopibicUzc7uPribsnS6o=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_net",
        importpath = "github.com/hashicorp/go.net",
        sum = "h1:sNCoNyDEvN1xa+X0baata4RdcpKwcMS6DH+xwfqPgjw=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_hashicorp_go_rootcerts",
        importpath = "github.com/hashicorp/go-rootcerts",
        sum = "h1:Rqb66Oo1X/eSV1x66xbDccZjhJigjg0+e82kpwzSwCI=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_sockaddr",
        importpath = "github.com/hashicorp/go-sockaddr",
        sum = "h1:GeH6tui99pF4NJgfnhp+L6+FfobzVW3Ah46sLo0ICXs=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_go_syslog",
        importpath = "github.com/hashicorp/go-syslog",
        sum = "h1:KaodqZuhUoZereWVIYmpUgZysurB1kBLX2j0MwMrUAE=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_hashicorp_go_uuid",
        importpath = "github.com/hashicorp/go-uuid",
        sum = "h1:cfejS+Tpcp13yd5nYHWDI6qVCny6wyX2Mt5SGur2IGE=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_hashicorp_go_version",
        importpath = "github.com/hashicorp/go-version",
        sum = "h1:zEfKbn2+PDgroKdiOzqiE8rsmLqU2uwi5PB5pBJ3TkI=",
        version = "v1.2.1",
    )

    go_repository(
        name = "com_github_hashicorp_golang_lru",
        importpath = "github.com/hashicorp/golang-lru",
        sum = "h1:dg1dEPuWpEqDnvIw251EVy4zlP8gWbsGj4BsUKCRpYs=",
        version = "v0.5.5-0.20210104140557-80c98217689d",
    )
    go_repository(
        name = "com_github_hashicorp_hcl",
        importpath = "github.com/hashicorp/hcl",
        sum = "h1:0Anlzjpi4vEasTeNFn2mLJgTSwt0+6sfsiTG8qcWGx4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_logutils",
        importpath = "github.com/hashicorp/logutils",
        sum = "h1:dLEQVugN8vlakKOUE3ihGLTZJRB4j+M2cdTm/ORI65Y=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_mdns",
        importpath = "github.com/hashicorp/mdns",
        sum = "h1:WhIgCr5a7AaVH6jPUwjtRuuE7/RDufnUvzIr48smyxs=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hashicorp_memberlist",
        importpath = "github.com/hashicorp/memberlist",
        sum = "h1:EmmoJme1matNzb+hMpDuR/0sbJSUisxyqBGG676r31M=",
        version = "v0.1.3",
    )
    go_repository(
        name = "com_github_hashicorp_serf",
        importpath = "github.com/hashicorp/serf",
        sum = "h1:YZ7UKsJv+hKjqGVUUbtE3HNj79Eln2oQ75tniF6iPt0=",
        version = "v0.8.2",
    )

    go_repository(
        name = "com_github_herumi_bls_eth_go_binary",
        importpath = "github.com/herumi/bls-eth-go-binary",
        sum = "h1:wCMygKUQhmcQAjlk2Gquzq6dLmyMv2kF+llRspoRgrk=",
        version = "v0.0.0-20210917013441-d37c07cfda4e",
    )

    go_repository(
        name = "com_github_holiman_bloomfilter_v2",
        importpath = "github.com/holiman/bloomfilter/v2",
        sum = "h1:73e0e/V0tCydx14a0SCYS/EWCxgwLZ18CZcZKVu0fao=",
        version = "v2.0.3",
    )
    go_repository(
        name = "com_github_holiman_goevmlab",
        importpath = "github.com/holiman/goevmlab",
        sum = "h1:VwUWx8Yz53Ch/vYauEr4PM//bFWSeivjP5HVgPQuz00=",
        version = "v0.0.0-20211215113238-06157bc85f7d",
    )

    go_repository(
        name = "com_github_holiman_uint256",
        importpath = "github.com/holiman/uint256",
        sum = "h1:gpSYcPLWGv4sG43I2mVLiDZCNDh/EpGjSk8tmtxitHM=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_hpcloud_tail",
        importpath = "github.com/hpcloud/tail",
        sum = "h1:nfCOvKYfkgYP8hkirhJocXT2+zOD8yUNjXaWfTlyFKI=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_hudl_fargo",
        importpath = "github.com/hudl/fargo",
        sum = "h1:0U6+BtN6LhaYuTnIJq4Wyq5cpn6O2kWrxAtcqBmYY6w=",
        version = "v1.3.0",
    )

    go_repository(
        name = "com_github_huin_goupnp",
        importpath = "github.com/huin/goupnp",
        sum = "h1:N8No57ls+MnjlB+JPiCVSOyy/ot7MJTqlo7rn+NYSqQ=",
        version = "v1.0.3",
    )
    go_repository(
        name = "com_github_huin_goutil",
        importpath = "github.com/huin/goutil",
        sum = "h1:vlNjIqmUZ9CMAWsbURYl3a6wZbw7q5RHVvlXTNS/Bs8=",
        version = "v0.0.0-20170803182201-1ca381bf3150",
    )

    go_repository(
        name = "com_github_ianlancetaylor_cgosymbolizer",
        importpath = "github.com/ianlancetaylor/cgosymbolizer",
        sum = "h1:IpTHAzWv1pKDDWeJDY5VOHvqc2T9d3C8cPKEf2VPqHE=",
        version = "v0.0.0-20200424224625-be1b05b0b279",
    )
    go_repository(
        name = "com_github_ianlancetaylor_demangle",
        importpath = "github.com/ianlancetaylor/demangle",
        sum = "h1:mV02weKRL81bEnm8A0HT1/CAelMQDBuQIfLw8n+d6xI=",
        version = "v0.0.0-20200824232613-28f6c0f3b639",
    )

    go_repository(
        name = "com_github_imdario_mergo",
        importpath = "github.com/imdario/mergo",
        sum = "h1:JboBksRwiiAJWvIYJVo46AfV+IAIKZpfrSzVKj42R4Q=",
        version = "v0.3.5",
    )
    go_repository(
        name = "com_github_inconshreveable_log15",
        importpath = "github.com/inconshreveable/log15",
        sum = "h1:g/SJtZVYc1cxSB8lgrgqeOlIdi4MhqNNHYRAC8y+g4c=",
        version = "v0.0.0-20170622235902-74a0988b5f80",
    )

    go_repository(
        name = "com_github_inconshreveable_mousetrap",
        importpath = "github.com/inconshreveable/mousetrap",
        sum = "h1:Z8tu5sraLXCXIcARxBp/8cbvlwVa7Z1NHg9XEKhtSvM=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_influxdata_flux",
        importpath = "github.com/influxdata/flux",
        sum = "h1:77BcVUCzvN5HMm8+j9PRBQ4iZcu98Dl4Y9rf+J5vhnc=",
        version = "v0.65.1",
    )
    go_repository(
        name = "com_github_influxdata_influxdb",
        importpath = "github.com/influxdata/influxdb",
        sum = "h1:WEypI1BQFTT4teLM+1qkEcvUi0dAvopAI/ir0vAiBg8=",
        version = "v1.8.3",
    )
    go_repository(
        name = "com_github_influxdata_influxdb1_client",
        importpath = "github.com/influxdata/influxdb1-client",
        sum = "h1:/WZQPMZNsjZ7IlCpsLGdQBINg5bxKQ1K1sh6awxLtkA=",
        version = "v0.0.0-20191209144304-8bf82d3c094d",
    )
    go_repository(
        name = "com_github_influxdata_influxdb_client_go_v2",
        importpath = "github.com/influxdata/influxdb-client-go/v2",
        sum = "h1:HGBfZYStlx3Kqvsv1h2pJixbCl/jhnFtxpKFAv9Tu5k=",
        version = "v2.4.0",
    )

    go_repository(
        name = "com_github_influxdata_influxql",
        importpath = "github.com/influxdata/influxql",
        sum = "h1:ED4e5Cc3z5vSN2Tz2GkOHN7vs4Sxe2yds6CXvDnvZFE=",
        version = "v1.1.1-0.20200828144457-65d3ef77d385",
    )
    go_repository(
        name = "com_github_influxdata_line_protocol",
        importpath = "github.com/influxdata/line-protocol",
        sum = "h1:vilfsDSy7TDxedi9gyBkMvAirat/oRcL0lFdJBf6tdM=",
        version = "v0.0.0-20210311194329-9aa0e372d097",
    )
    go_repository(
        name = "com_github_influxdata_promql_v2",
        importpath = "github.com/influxdata/promql/v2",
        sum = "h1:kXn3p0D7zPw16rOtfDR+wo6aaiH8tSMfhPwONTxrlEc=",
        version = "v2.12.0",
    )
    go_repository(
        name = "com_github_influxdata_roaring",
        importpath = "github.com/influxdata/roaring",
        sum = "h1:UzJnB7VRL4PSkUJHwsyzseGOmrO/r4yA+AuxGJxiZmA=",
        version = "v0.4.13-0.20180809181101-fc520f41fab6",
    )
    go_repository(
        name = "com_github_influxdata_tdigest",
        importpath = "github.com/influxdata/tdigest",
        sum = "h1:MHTrDWmQpHq/hkq+7cw9oYAt2PqUw52TZazRA0N7PGE=",
        version = "v0.0.0-20181121200506-bf2b5ad3c0a9",
    )
    go_repository(
        name = "com_github_influxdata_usage_client",
        importpath = "github.com/influxdata/usage-client",
        sum = "h1:+TUUmaFa4YD1Q+7bH9o5NCHQGPMqZCYJiNW6lIIS9z4=",
        version = "v0.0.0-20160829180054-6d3895376368",
    )
    go_repository(
        name = "com_github_ipfs_go_cid",
        importpath = "github.com/ipfs/go-cid",
        sum = "h1:01JTiihFq9en9Vz0lc0VDWvZe/uBonGpzo4THP0vcQ0=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_ipfs_go_datastore",
        importpath = "github.com/ipfs/go-datastore",
        sum = "h1:WkRhLuISI+XPD0uk3OskB0fYFSyqK8Ob5ZYew9Qa1nQ=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_ipfs_go_detect_race",
        importpath = "github.com/ipfs/go-detect-race",
        sum = "h1:qX/xay2W3E4Q1U7d9lNs1sU9nvguX0a7319XbyQ6cOk=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_badger",
        importpath = "github.com/ipfs/go-ds-badger",
        sum = "h1:xREL3V0EH9S219kFFueOYJJTcjgNSZ2HY1iSvN7U1Ro=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_leveldb",
        importpath = "github.com/ipfs/go-ds-leveldb",
        sum = "h1:s++MEBbD3ZKc9/8/njrn4flZLnCuY9I79v94gBUNumo=",
        version = "v0.5.0",
    )

    go_repository(
        name = "com_github_ipfs_go_ipfs_delay",
        importpath = "github.com/ipfs/go-ipfs-delay",
        sum = "h1:NAviDvJ0WXgD+yiL2Rj35AmnfgI11+pHXbdciD917U0=",
        version = "v0.0.0-20181109222059-70721b86a9a8",
    )

    go_repository(
        name = "com_github_ipfs_go_ipfs_util",
        importpath = "github.com/ipfs/go-ipfs-util",
        sum = "h1:59Sswnk1MFaiq+VcaknX7aYEyGyGDAA73ilhEK2POp8=",
        version = "v0.0.2",
    )

    go_repository(
        name = "com_github_ipfs_go_log",
        build_file_proto_mode = "disable_global",
        importpath = "github.com/ipfs/go-log",
        sum = "h1:2dOuUCB1Z7uoczMWgAyDck5JLb72zHzrMnGnCNNbvY8=",
        version = "v1.0.5",
    )
    go_repository(
        name = "com_github_ipfs_go_log_v2",
        build_file_proto_mode = "disable_global",
        importpath = "github.com/ipfs/go-log/v2",
        sum = "h1:1XdUzF7048prq4aBjDQQ4SL5RxftpRGdXhNRwKSAlcY=",
        version = "v2.5.1",
    )

    go_repository(
        name = "com_github_jackpal_go_nat_pmp",
        importpath = "github.com/jackpal/go-nat-pmp",
        sum = "h1:KzKSgb7qkJvOUTqYl9/Hg/me3pWgBmERKrTGD7BdWus=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_jbenet_go_cienv",
        importpath = "github.com/jbenet/go-cienv",
        sum = "h1:Vc/s0QbQtoxX8MwwSLWWh+xNNZvM3Lw7NsTcHrvvhMc=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_jbenet_go_temp_err_catcher",
        importpath = "github.com/jbenet/go-temp-err-catcher",
        sum = "h1:zpb3ZH6wIE8Shj2sKS+khgRvf7T7RABoLk/+KKHggpk=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_jbenet_goprocess",
        importpath = "github.com/jbenet/goprocess",
        sum = "h1:DRGOFReOMqqDNXwW70QkacFW0YN9QnwLV0Vqk+3oU0o=",
        version = "v0.1.4",
    )
    go_repository(
        name = "com_github_jcmturner_gofork",
        importpath = "github.com/jcmturner/gofork",
        sum = "h1:J7uCkflzTEhUZ64xqKnkDxq3kzc96ajM1Gli5ktUem8=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_jedisct1_go_minisign",
        importpath = "github.com/jedisct1/go-minisign",
        sum = "h1:UvSe12bq+Uj2hWd8aOlwPmoZ+CITRFrdit+sDGfAg8U=",
        version = "v0.0.0-20190909160543-45766022959e",
    )
    go_repository(
        name = "com_github_jellevandenhooff_dkim",
        importpath = "github.com/jellevandenhooff/dkim",
        sum = "h1:ujPKutqRlJtcfWk6toYVYagwra7HQHbXOaS171b4Tg8=",
        version = "v0.0.0-20150330215556-f50fe3d243e1",
    )

    go_repository(
        name = "com_github_jessevdk_go_flags",
        importpath = "github.com/jessevdk/go-flags",
        sum = "h1:4IU2WS7AumrZ/40jfhf4QVDMsQwqA7VEHozFRrGARJA=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_jhump_protoreflect",
        importpath = "github.com/jhump/protoreflect",
        sum = "h1:z7Ciiz3Bz37zSd485fbiTW8ABafIasyOWZI0N9EUUdo=",
        version = "v1.8.1",
    )
    go_repository(
        name = "com_github_jmespath_go_jmespath",
        importpath = "github.com/jmespath/go-jmespath",
        sum = "h1:BEgLn5cpjn8UN1mAw4NjwDrS35OdebyEtFe+9YPoQUg=",
        version = "v0.4.0",
    )

    go_repository(
        name = "com_github_jmespath_go_jmespath_internal_testify",
        importpath = "github.com/jmespath/go-jmespath/internal/testify",
        sum = "h1:shLQSRRSCCPj3f2gpwzGwWFoC7ycTf1rcQZHOlsJ6N8=",
        version = "v1.5.1",
    )
    go_repository(
        name = "com_github_jonboulle_clockwork",
        importpath = "github.com/jonboulle/clockwork",
        sum = "h1:VKV+ZcuP6l3yW9doeqz6ziZGgcynBVQO+obU0+0hcPo=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_joonix_log",
        importpath = "github.com/joonix/log",
        sum = "h1:k+SfYbN66Ev/GDVq39wYOXVW5RNd5kzzairbCe9dK5Q=",
        version = "v0.0.0-20200409080653-9c1d2ceb5f1d",
    )
    go_repository(
        name = "com_github_jpillora_backoff",
        importpath = "github.com/jpillora/backoff",
        sum = "h1:uvFg412JmmHBHw7iwprIxkPMI+sGQ4kzOWsMeHnm2EA=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_jrick_logrotate",
        importpath = "github.com/jrick/logrotate",
        sum = "h1:lQ1bL/n9mBNeIXoTUoYRlK4dHuNJVofX9oWqBtPnSzI=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_json_iterator_go",
        importpath = "github.com/json-iterator/go",
        replace = "github.com/prestonvanloon/go",
        sum = "h1:Bt5PzQCqfP4xiLXDSrMoqAfj6CBr3N9DAyyq8OiIWsc=",
        version = "v1.1.7-0.20190722034630-4f2e55fcf87b",
    )
    go_repository(
        name = "com_github_jstemmer_go_junit_report",
        importpath = "github.com/jstemmer/go-junit-report",
        sum = "h1:6QPYqodiu3GuPL+7mfx+NwDdp2eTkp9IfEUpgAwUN0o=",
        version = "v0.9.1",
    )

    go_repository(
        name = "com_github_jsternberg_zap_logfmt",
        importpath = "github.com/jsternberg/zap-logfmt",
        sum = "h1:0Dz2s/eturmdUS34GM82JwNEdQ9hPoJgqptcEKcbpzY=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_jtolds_gls",
        importpath = "github.com/jtolds/gls",
        sum = "h1:xdiiI2gbIgH/gLH7ADydsJ1uDOEzR8yvV7C0MuV77Wo=",
        version = "v4.20.0+incompatible",
    )

    go_repository(
        name = "com_github_juju_ansiterm",
        importpath = "github.com/juju/ansiterm",
        sum = "h1:FaWFmfWdAUKbSCtOU2QjDaorUexogfaMgbipgYATUMU=",
        version = "v0.0.0-20180109212912-720a0952cc2a",
    )
    go_repository(
        name = "com_github_julienschmidt_httprouter",
        importpath = "github.com/julienschmidt/httprouter",
        sum = "h1:U0609e9tgbseu3rBINet9P48AI/D3oJs4dN7jwJOQ1U=",
        version = "v1.3.0",
    )

    go_repository(
        name = "com_github_jung_kurt_gofpdf",
        importpath = "github.com/jung-kurt/gofpdf",
        sum = "h1:PJr+ZMXIecYc1Ey2zucXdR73SMBtgjPgwa31099IMv0=",
        version = "v1.0.3-0.20190309125859-24315acbbda5",
    )
    go_repository(
        name = "com_github_jwilder_encoding",
        importpath = "github.com/jwilder/encoding",
        sum = "h1:2jNeR4YUziVtswNP9sEFAI913cVrzH85T+8Q6LpYbT0=",
        version = "v0.0.0-20170811194829-b4e1701a28ef",
    )

    go_repository(
        name = "com_github_k0kubun_go_ansi",
        importpath = "github.com/k0kubun/go-ansi",
        sum = "h1:qGQQKEcAR99REcMpsXCp3lJ03zYT1PkRd3kQGPn9GVg=",
        version = "v0.0.0-20180517002512-3bf9e2903213",
    )
    go_repository(
        name = "com_github_kami_zh_go_capturer",
        importpath = "github.com/kami-zh/go-capturer",
        sum = "h1:cVtBfNW5XTHiKQe7jDaDBSh/EVM4XLPutLAGboIXuM0=",
        version = "v0.0.0-20171211120116-e492ea43421d",
    )
    go_repository(
        name = "com_github_karalabe_usb",
        importpath = "github.com/karalabe/usb",
        sum = "h1:M6QQBNxF+CQ8OFvxrT90BA0qBOXymndZnk5q235mFc4=",
        version = "v0.0.2",
    )

    go_repository(
        name = "com_github_kevinms_leakybucket_go",
        importpath = "github.com/kevinms/leakybucket-go",
        sum = "h1:qNtd6alRqd3qOdPrKXMZImV192ngQ0WSh1briEO33Tk=",
        version = "v0.0.0-20200115003610-082473db97ca",
    )
    go_repository(
        name = "com_github_kilic_bls12_381",
        importpath = "github.com/kilic/bls12-381",
        sum = "h1:eZB80d/IKkIPjCTLUBT6+Imzn2zLpXtJFzY986jlHV4=",
        version = "v0.0.0-20201226121925-69dacb279461",
    )

    go_repository(
        name = "com_github_kisielk_errcheck",
        importpath = "github.com/kisielk/errcheck",
        sum = "h1:e8esj/e4R+SAOwFwN+n3zr0nYeCyeweozKfO23MvHzY=",
        version = "v1.5.0",
    )
    go_repository(
        name = "com_github_kisielk_gotool",
        importpath = "github.com/kisielk/gotool",
        sum = "h1:AV2c/EiW3KqPNT9ZKl07ehoAGi4C5/01Cfbblndcapg=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_kkdai_bstream",
        importpath = "github.com/kkdai/bstream",
        sum = "h1:FOOIBWrEkLgmlgGfMuZT83xIwfPDxEI2OHu6xUmJMFE=",
        version = "v0.0.0-20161212061736-f391b8402d23",
    )
    go_repository(
        name = "com_github_klauspost_compress",
        importpath = "github.com/klauspost/compress",
        sum = "h1:7cgTQxJCU/vy+oP/E3B9RGbQTgbiVzIJWIKOLoAsPok=",
        version = "v1.15.7",
    )

    go_repository(
        name = "com_github_klauspost_cpuid",
        importpath = "github.com/klauspost/cpuid",
        sum = "h1:CCtW0xUnWGVINKvE/WWOYKdsPV6mawAtvQuSl8guwQs=",
        version = "v1.2.3",
    )

    go_repository(
        name = "com_github_klauspost_cpuid_v2",
        importpath = "github.com/klauspost/cpuid/v2",
        sum = "h1:QRqdp6bb9M9S5yyKeYteXKuoKE4p0tGlra81fKOpWH8=",
        version = "v2.0.14",
    )
    go_repository(
        name = "com_github_klauspost_crc32",
        importpath = "github.com/klauspost/crc32",
        sum = "h1:KAZ1BW2TCmT6PRihDPpocIy1QTtsAsrx6TneU/4+CMg=",
        version = "v0.0.0-20161016154125-cb6bfca970f6",
    )
    go_repository(
        name = "com_github_klauspost_pgzip",
        importpath = "github.com/klauspost/pgzip",
        sum = "h1:qnWYvvKqedOF2ulHpMG72XQol4ILEJ8k2wwRl/Km8oE=",
        version = "v1.2.5",
    )
    go_repository(
        name = "com_github_klauspost_reedsolomon",
        importpath = "github.com/klauspost/reedsolomon",
        sum = "h1:N/VzgeMfHmLc+KHMD1UL/tNkfXAt8FnUqlgXGIduwAY=",
        version = "v1.9.3",
    )
    go_repository(
        name = "com_github_knetic_govaluate",
        importpath = "github.com/Knetic/govaluate",
        sum = "h1:1G1pk05UrOh0NlF1oeaaix1x8XzrfjIDK47TY0Zehcw=",
        version = "v3.0.1-0.20171022003610-9aa49832a739+incompatible",
    )

    go_repository(
        name = "com_github_konsorten_go_windows_terminal_sequences",
        importpath = "github.com/konsorten/go-windows-terminal-sequences",
        sum = "h1:CE8S1cTafDpPvMhIxNJKvHsGVBgn1xWYf1NbHQhywc8=",
        version = "v1.0.3",
    )
    go_repository(
        name = "com_github_koron_go_ssdp",
        importpath = "github.com/koron/go-ssdp",
        sum = "h1:JivLMY45N76b4p/vsWGOKewBQu6uf39y8l+AQ7sDKx8=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_korovkin_limiter",
        importpath = "github.com/korovkin/limiter",
        sum = "h1:QwKnpk6xFW80HVFKqiIHTzK19UF62mRWejcUr/q6z4I=",
        version = "v0.0.0-20190919045942-dac5a6b2a536",
    )

    go_repository(
        name = "com_github_kr_logfmt",
        importpath = "github.com/kr/logfmt",
        sum = "h1:T+h1c/A9Gawja4Y9mFVWj2vyii2bbUNDw3kt9VxK2EY=",
        version = "v0.0.0-20140226030751-b84e30acd515",
    )
    go_repository(
        name = "com_github_kr_pretty",
        importpath = "github.com/kr/pretty",
        sum = "h1:WgNl7dwNpEZ6jJ9k1snq4pZsg7DOEN8hP9Xw0Tsjwk0=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_kr_pty",
        importpath = "github.com/kr/pty",
        sum = "h1:/Um6a/ZmD5tF7peoOJ5oN5KMQ0DrGVQSXLNwyckutPk=",
        version = "v1.1.3",
    )
    go_repository(
        name = "com_github_kr_text",
        importpath = "github.com/kr/text",
        sum = "h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=",
        version = "v0.2.0",
    )

    go_repository(
        name = "com_github_kylelemons_godebug",
        importpath = "github.com/kylelemons/godebug",
        sum = "h1:RPNrshWIDI6G2gRW9EHilWtl7Z6Sb1BR0xunSBf0SNc=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_labstack_echo_v4",
        importpath = "github.com/labstack/echo/v4",
        sum = "h1:LF5Iq7t/jrtUuSutNuiEWtB5eiHfZ5gSe2pcu5exjQw=",
        version = "v4.2.1",
    )
    go_repository(
        name = "com_github_labstack_gommon",
        importpath = "github.com/labstack/gommon",
        sum = "h1:JEeO0bvc78PKdyHxloTKiF8BD5iGrH8T6MSeGvSgob0=",
        version = "v0.3.0",
    )

    go_repository(
        name = "com_github_leanovate_gopter",
        importpath = "github.com/leanovate/gopter",
        sum = "h1:fQjYxZaynp97ozCzfOyOuAGOU4aU/z37zf/tOujFk7c=",
        version = "v0.2.9",
    )
    go_repository(
        name = "com_github_leodido_go_urn",
        importpath = "github.com/leodido/go-urn",
        sum = "h1:BqpAaACuzVSgi/VLzGZIobT2z4v53pjosyNd9Yv6n/w=",
        version = "v1.2.1",
    )

    go_repository(
        name = "com_github_lib_pq",
        importpath = "github.com/lib/pq",
        sum = "h1:X5PMW56eZitiTeO7tKzZxFCSpbFZJtkMMooicw2us9A=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_libp2p_go_buffer_pool",
        importpath = "github.com/libp2p/go-buffer-pool",
        sum = "h1:oK4mSFcQz7cTQIfqbe4MIj9gLW+mnanjyFtc6cdF0Y8=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_libp2p_go_cidranger",
        importpath = "github.com/libp2p/go-cidranger",
        sum = "h1:ewPN8EZ0dd1LSnrtuwd4709PXVcITVeuwbag38yPW7c=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_libp2p_go_conn_security_multistream",
        importpath = "github.com/libp2p/go-conn-security-multistream",
        sum = "h1:9UCIKlBL1hC9u7nkMXpD1nkc/T53PKMAn3/k9ivBAVc=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_libp2p_go_eventbus",
        importpath = "github.com/libp2p/go-eventbus",
        sum = "h1:VanAdErQnpTioN2TowqNcOijf6YwhuODe4pPKSDpxGc=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_libp2p_go_flow_metrics",
        importpath = "github.com/libp2p/go-flow-metrics",
        sum = "h1:8tAs/hSdNvUiLgtlSy3mxwxWP4I9y/jlkPFT7epKdeM=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p",
        build_file_proto_mode = "disable_global",
        importpath = "github.com/libp2p/go-libp2p",
        sum = "h1:tjjDNfp7FqdI/7v1rXtB/BtELaPlAThL2uzlj18kcrw=",
        version = "v0.20.3",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_asn_util",
        importpath = "github.com/libp2p/go-libp2p-asn-util",
        sum = "h1:rg3+Os8jbnO5DxkC7K/Utdi+DkY3q/d1/1q+8WeNAsw=",
        version = "v0.2.0",
    )

    go_repository(
        name = "com_github_libp2p_go_libp2p_blankhost",
        importpath = "github.com/libp2p/go-libp2p-blankhost",
        sum = "h1:3EsGAi0CBGcZ33GwRuXEYJLLPoVWyXJ1bcJzAJjINkk=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_circuit",
        build_file_proto_mode = "disable_global",
        importpath = "github.com/libp2p/go-libp2p-circuit",
        sum = "h1:rw/HlhmUB3OktS/Ygz6+2XABOmHKzZpPUuMNUMosj8w=",
        version = "v0.6.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_connmgr",
        importpath = "github.com/libp2p/go-libp2p-connmgr",
        sum = "h1:TMS0vc0TCBomtQJyWr7fYxcVYYhx+q/2gF++G5Jkl/w=",
        version = "v0.2.4",
    )

    go_repository(
        name = "com_github_libp2p_go_libp2p_core",
        build_directives = [
            "gazelle:resolve go github.com/btcsuite/btcd/btcec @com_github_btcsuite_btcd//btcec:btcec",
        ],
        build_file_proto_mode = "disable_global",
        importpath = "github.com/libp2p/go-libp2p-core",
        sum = "h1:QGU8mlxHytwTc4pq/aVQX9VDoAPiCHxfe/oOSwF+YDg=",
        version = "v0.17.0",
    )

    go_repository(
        name = "com_github_libp2p_go_libp2p_mplex",
        importpath = "github.com/libp2p/go-libp2p-mplex",
        sum = "h1:vt3k4E4HSND9XH4Z8rUpacPJFSAgLOv6HDvG8W9Ks9E=",
        version = "v0.5.0",
    )

    go_repository(
        name = "com_github_libp2p_go_libp2p_peerstore",
        importpath = "github.com/libp2p/go-libp2p-peerstore",
        sum = "h1:2iIUwok3vtmnWJTZeTeLgnBO6GbkXcwSRwgZHEKrQZs=",
        version = "v0.7.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_pnet",
        importpath = "github.com/libp2p/go-libp2p-pnet",
        sum = "h1:J6htxttBipJujEjz1y0a5+eYoiPcFHhSYHH6na5f0/k=",
        version = "v0.2.0",
    )

    go_repository(
        name = "com_github_libp2p_go_libp2p_pubsub",
        build_file_proto_mode = "disable_global",
        importpath = "github.com/libp2p/go-libp2p-pubsub",
        sum = "h1:Om2zX4v6lQo7CIT99xkCVTSXeSwjfPtzzN3nzQRlApA=",
        version = "v0.7.1-0.20220701163738-60cf38003244",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_quic_transport",
        importpath = "github.com/libp2p/go-libp2p-quic-transport",
        sum = "h1:aVg9/jr+R2esov5sH7wkXrmYmqJiUjtLMLYX3L9KYdY=",
        version = "v0.16.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_resource_manager",
        importpath = "github.com/libp2p/go-libp2p-resource-manager",
        sum = "h1:2+cYxUNi33tcydsVLt6K5Fv2E3OTiVeafltecAj15E0=",
        version = "v0.3.0",
    )

    go_repository(
        name = "com_github_libp2p_go_libp2p_swarm",
        build_file_proto_mode = "disable_global",
        importpath = "github.com/libp2p/go-libp2p-swarm",
        sum = "h1:1yr7UCwxCN92cw9g9Q+fnJSlk7lOB1RetoEewxhGVL0=",
        version = "v0.10.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_testing",
        importpath = "github.com/libp2p/go-libp2p-testing",
        sum = "h1:dCpODRtRaDZKF8HXT9qqqgON+OMEB423Knrgeod8j84=",
        version = "v0.9.2",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_tls",
        importpath = "github.com/libp2p/go-libp2p-tls",
        sum = "h1:8BgvUJiOTcj0Gp6XvEicF0rL5aUtRg/UzEdeZDmDlC8=",
        version = "v0.3.0",
    )

    go_repository(
        name = "com_github_libp2p_go_libp2p_transport_upgrader",
        importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
        sum = "h1:ADnLrL7fC4Vy7HPjk9oGof7nDeTqGXuof85Ar6kin9Q=",
        version = "v0.7.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_yamux",
        importpath = "github.com/libp2p/go-libp2p-yamux",
        sum = "h1:APQYlttIj+Rr5sfa6siojwsi0ZwcIh/exHIUl9hZr6o=",
        version = "v0.8.0",
    )
    go_repository(
        name = "com_github_libp2p_go_maddr_filter",
        importpath = "github.com/libp2p/go-maddr-filter",
        sum = "h1:4ACqZKw8AqiuJfwFGq1CYDFugfXTOos+qQ3DETkhtCE=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_libp2p_go_mplex",
        importpath = "github.com/libp2p/go-mplex",
        sum = "h1:BDhFZdlk5tbr0oyFq/xv/NPGfjbnrsDam1EvutpBDbY=",
        version = "v0.7.0",
    )
    go_repository(
        name = "com_github_libp2p_go_msgio",
        importpath = "github.com/libp2p/go-msgio",
        sum = "h1:W6shmB+FeynDrUVl2dgFQvzfBZcXiyqY4VmpQLu9FqU=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_libp2p_go_nat",
        importpath = "github.com/libp2p/go-nat",
        sum = "h1:MfVsH6DLcpa04Xr+p8hmVRG4juse0s3J8HyNWYHffXg=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_libp2p_go_netroute",
        importpath = "github.com/libp2p/go-netroute",
        sum = "h1:0FpsbsvuSnAhXFnCY0VLFbJOzaK0VnP0r1QT/o4nWRE=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_libp2p_go_openssl",
        importpath = "github.com/libp2p/go-openssl",
        sum = "h1:eCAzdLejcNVBzP/iZM9vqHnQm+XyCEbSSIheIPRGNsw=",
        version = "v0.0.7",
    )

    go_repository(
        name = "com_github_libp2p_go_reuseport",
        importpath = "github.com/libp2p/go-reuseport",
        sum = "h1:18PRvIMlpY6ZK85nIAicSBuXXvrYoSw3dsBAR7zc560=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_libp2p_go_reuseport_transport",
        importpath = "github.com/libp2p/go-reuseport-transport",
        sum = "h1:C3PHeHjmnz8m6f0uydObj02tMEoi7CyD1zuN7xQT8gc=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_libp2p_go_sockaddr",
        importpath = "github.com/libp2p/go-sockaddr",
        sum = "h1:Y4s3/jNoryVRKEBrkJ576F17CPOaMIzUeCsg7dlTDj0=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_libp2p_go_stream_muxer_multistream",
        importpath = "github.com/libp2p/go-stream-muxer-multistream",
        sum = "h1:HsM/9OdtqnIzjVXcxTXjmqKrj3gJ8kacaOJwJS1ipaY=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_libp2p_go_tcp_transport",
        importpath = "github.com/libp2p/go-tcp-transport",
        sum = "h1:3ZPW8HAuyRAuFzyabE0hSrCXKKSWzROnZZX7DtcIatY=",
        version = "v0.5.0",
    )

    go_repository(
        name = "com_github_libp2p_go_yamux_v3",
        importpath = "github.com/libp2p/go-yamux/v3",
        sum = "h1:lNEy28MBk1HavUAlzKgShp+F6mn/ea1nDYWftZhFW9Q=",
        version = "v3.1.2",
    )

    go_repository(
        name = "com_github_libp2p_zeroconf_v2",
        importpath = "github.com/libp2p/zeroconf/v2",
        sum = "h1:XAuSczA96MYkVwH+LqqqCUZb2yH3krobMJ1YE+0hG2s=",
        version = "v2.1.1",
    )

    go_repository(
        name = "com_github_lightstep_lightstep_tracer_common_golang_gogo",
        importpath = "github.com/lightstep/lightstep-tracer-common/golang/gogo",
        sum = "h1:143Bb8f8DuGWck/xpNUOckBVYfFbBTnLevfRZ1aVVqo=",
        version = "v0.0.0-20190605223551-bc2310a04743",
    )
    go_repository(
        name = "com_github_lightstep_lightstep_tracer_go",
        importpath = "github.com/lightstep/lightstep-tracer-go",
        sum = "h1:vi1F1IQ8N7hNWytK9DpJsUfQhGuNSc19z330K6vl4zk=",
        version = "v0.18.1",
    )

    go_repository(
        name = "com_github_logrusorgru_aurora",
        importpath = "github.com/logrusorgru/aurora",
        sum = "h1:tOpm7WcpBTn4fjmVfgpQq0EfczGlG91VSDkswnjF5A8=",
        version = "v2.0.3+incompatible",
    )
    go_repository(
        name = "com_github_lucas_clemente_quic_go",
        importpath = "github.com/lucas-clemente/quic-go",
        sum = "h1:zsMwwniyybb8B/UDNXRSYee7WpQJVOcjQEGgpw2ikXs=",
        version = "v0.27.2",
    )
    go_repository(
        name = "com_github_lucasb_eyer_go_colorful",
        importpath = "github.com/lucasb-eyer/go-colorful",
        sum = "h1:QIbQXiugsb+q10B+MI+7DI1oQLdmnep86tWFlaaUAac=",
        version = "v1.0.3",
    )

    go_repository(
        name = "com_github_lunixbochs_vtclean",
        importpath = "github.com/lunixbochs/vtclean",
        sum = "h1:xu2sLAri4lGiovBDQKxl5mrXyESr3gUr5m5SM5+LVb8=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_lyft_protoc_gen_validate",
        importpath = "github.com/lyft/protoc-gen-validate",
        sum = "h1:KNt/RhmQTOLr7Aj8PsJ7mTronaFyx80mRTT9qF261dA=",
        version = "v0.0.13",
    )

    go_repository(
        name = "com_github_magiconair_properties",
        importpath = "github.com/magiconair/properties",
        sum = "h1:ZC2Vc7/ZFkGmsVC9KvOjumD+G5lXy2RtTKyzRKO2BQ4=",
        version = "v1.8.1",
    )
    go_repository(
        name = "com_github_mailru_easyjson",
        importpath = "github.com/mailru/easyjson",
        sum = "h1:hB2xlXdHp/pmPZq0y3QnmWAArdw9PqbmotexnWx/FU8=",
        version = "v0.0.0-20190626092158-b2ccc519800e",
    )

    go_repository(
        name = "com_github_manifoldco_promptui",
        importpath = "github.com/manifoldco/promptui",
        sum = "h1:3l11YT8tm9MnwGFQ4kETwkzpAwY2Jt9lCrumCUW4+z4=",
        version = "v0.7.0",
    )
    go_repository(
        name = "com_github_mariusvanderwijden_fuzzyvm",
        importpath = "github.com/MariusVanDerWijden/FuzzyVM",
        sum = "h1:HKOeocqWNWitsHPVPdCUJRalFmDNVHs2xTKmse4svwU=",
        version = "v0.0.0-20220304110512-764253afa8c2",
    )
    go_repository(
        name = "com_github_mariusvanderwijden_tx_fuzz",
        importpath = "github.com/MariusVanDerWijden/tx-fuzz",
        sum = "h1:sVb99a9AUxf26t06VNnDO5DReatE0lXCp6amDOaalXM=",
        version = "v0.0.0-20220321065247-ebb195301a27",
    )

    go_repository(
        name = "com_github_marten_seemann_qpack",
        importpath = "github.com/marten-seemann/qpack",
        sum = "h1:jvTsT/HpCn2UZJdP+UUB53FfUUgeOyG5K1ns0OJOGVs=",
        version = "v0.2.1",
    )

    go_repository(
        name = "com_github_marten_seemann_qtls_go1_15",
        importpath = "github.com/marten-seemann/qtls-go1-15",
        sum = "h1:RehYMOyRW8hPVEja1KBVsFVNSm35Jj9Mvs5yNoZZ28A=",
        version = "v0.1.4",
    )
    go_repository(
        name = "com_github_marten_seemann_qtls_go1_16",
        importpath = "github.com/marten-seemann/qtls-go1-16",
        sum = "h1:o9JrYPPco/Nukd/HpOHMHZoBDXQqoNtUCmny98/1uqQ=",
        version = "v0.1.5",
    )
    go_repository(
        name = "com_github_marten_seemann_qtls_go1_17",
        importpath = "github.com/marten-seemann/qtls-go1-17",
        sum = "h1:JADBlm0LYiVbuSySCHeY863dNkcpMmDR7s0bLKJeYlQ=",
        version = "v0.1.2",
    )
    go_repository(
        name = "com_github_marten_seemann_qtls_go1_18",
        build_directives = [
            "gazelle:exclude generate_cert.go",
        ],
        importpath = "github.com/marten-seemann/qtls-go1-18",
        sum = "h1:JH6jmzbduz0ITVQ7ShevK10Av5+jBEKAHMntXmIV7kM=",
        version = "v0.1.2",
    )

    go_repository(
        name = "com_github_marten_seemann_tcp",
        importpath = "github.com/marten-seemann/tcp",
        sum = "h1:br0buuQ854V8u83wA0rVZ8ttrq5CpaPZdvrK0LP2lOk=",
        version = "v0.0.0-20210406111302-dfbc87cc63fd",
    )
    go_repository(
        name = "com_github_matryer_moq",
        importpath = "github.com/matryer/moq",
        sum = "h1:HvFwW+cm9bCbZ/+vuGNq7CRWXql8c0y8nGeYpqmpvmk=",
        version = "v0.0.0-20190312154309-6cfb0558e1bd",
    )

    go_repository(
        name = "com_github_mattn_go_colorable",
        importpath = "github.com/mattn/go-colorable",
        sum = "h1:c1ghPdyEDarC70ftn0y+A/Ee++9zz8ljHG1b13eJ0s8=",
        version = "v0.1.8",
    )
    go_repository(
        name = "com_github_mattn_go_ieproxy",
        importpath = "github.com/mattn/go-ieproxy",
        sum = "h1:oNAwILwmgWKFpuU+dXvI6dl9jG2mAWAZLX3r9s0PPiw=",
        version = "v0.0.0-20190702010315-6dee0af9227d",
    )

    go_repository(
        name = "com_github_mattn_go_isatty",
        importpath = "github.com/mattn/go-isatty",
        sum = "h1:yVuAays6BHfxijgZPzw+3Zlu5yQgKGP2/hcQbHb7S9Y=",
        version = "v0.0.14",
    )
    go_repository(
        name = "com_github_mattn_go_runewidth",
        importpath = "github.com/mattn/go-runewidth",
        sum = "h1:Lm995f3rfxdpd6TSmuVCHVb/QhupuXlYr8sCI/QdE+0=",
        version = "v0.0.9",
    )

    go_repository(
        name = "com_github_mattn_go_sqlite3",
        importpath = "github.com/mattn/go-sqlite3",
        sum = "h1:LDdKkqtYlom37fkvqs8rMPFKAMe8+SgjbwZ6ex1/A/Q=",
        version = "v1.11.0",
    )
    go_repository(
        name = "com_github_mattn_go_tty",
        importpath = "github.com/mattn/go-tty",
        sum = "h1:d8RFOZ2IiFtFWBcKEHAFYJcPTf0wY5q0exFNJZVWa1U=",
        version = "v0.0.0-20180907095812-13ff1204f104",
    )
    go_repository(
        name = "com_github_matttproud_golang_protobuf_extensions",
        importpath = "github.com/matttproud/golang_protobuf_extensions",
        sum = "h1:4hp9jkHxhMHkqkrB3Ix0jegS5sx/RkqARlsWZ6pIwiU=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_mgutz_ansi",
        importpath = "github.com/mgutz/ansi",
        sum = "h1:j7+1HpAFS1zy5+Q4qx1fWh90gTKwiN4QCGoY9TWyyO4=",
        version = "v0.0.0-20170206155736-9520e82c474b",
    )
    go_repository(
        name = "com_github_microcosm_cc_bluemonday",
        importpath = "github.com/microcosm-cc/bluemonday",
        sum = "h1:SIYunPjnlXcW+gVfvm0IlSeR5U3WZUOLfVmqg85Go44=",
        version = "v1.0.1",
    )

    go_repository(
        name = "com_github_miekg_dns",
        importpath = "github.com/miekg/dns",
        sum = "h1:DQUfb9uc6smULcREF09Uc+/Gd46YWqJd5DbpPE9xkcA=",
        version = "v1.1.50",
    )
    go_repository(
        name = "com_github_mikioh_tcp",
        importpath = "github.com/mikioh/tcp",
        sum = "h1:bzE/A84HN25pxAuk9Eej1Kz9OUelF97nAc82bDquQI8=",
        version = "v0.0.0-20190314235350-803a9b46060c",
    )
    go_repository(
        name = "com_github_mikioh_tcpinfo",
        importpath = "github.com/mikioh/tcpinfo",
        sum = "h1:z78hV3sbSMAUoyUMM0I83AUIT6Hu17AWfgjzIbtrYFc=",
        version = "v0.0.0-20190314235526-30a79bb1804b",
    )
    go_repository(
        name = "com_github_mikioh_tcpopt",
        importpath = "github.com/mikioh/tcpopt",
        sum = "h1:PTfri+PuQmWDqERdnNMiD9ZejrlswWrCpBEZgWOiTrc=",
        version = "v0.0.0-20190314235656-172688c1accc",
    )

    go_repository(
        name = "com_github_minio_blake2b_simd",
        importpath = "github.com/minio/blake2b-simd",
        sum = "h1:lYpkrQH5ajf0OXOcUbGjvZxxijuBwbbmlSxLiuofa+g=",
        version = "v0.0.0-20160723061019-3f5f724cb5b1",
    )
    go_repository(
        name = "com_github_minio_highwayhash",
        importpath = "github.com/minio/highwayhash",
        sum = "h1:dZ6IIu8Z14VlC0VpfKofAhCy74wu/Qb5gcn52yWoz/0=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_minio_sha256_simd",
        importpath = "github.com/minio/sha256-simd",
        sum = "h1:v1ta+49hkWZyvaKwrQB8elexRqm6Y0aMLjCNsrYxo6g=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_cli",
        importpath = "github.com/mitchellh/cli",
        sum = "h1:iGBIsUe3+HZ/AD/Vd7DErOt5sU9fa8Uj7A2s1aggv1Y=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_mitchellh_colorstring",
        importpath = "github.com/mitchellh/colorstring",
        sum = "h1:62I3jR2EmQ4l5rM/4FEfDWcRD+abF5XlKShorW5LRoQ=",
        version = "v0.0.0-20190213212951-d06e56a500db",
    )

    go_repository(
        name = "com_github_mitchellh_go_homedir",
        importpath = "github.com/mitchellh/go-homedir",
        sum = "h1:lukF9ziXFxDFPkA1vsr5zpc1XuPDn/wFntq5mG+4E0Y=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_mitchellh_go_testing_interface",
        importpath = "github.com/mitchellh/go-testing-interface",
        sum = "h1:fzU/JVNcaqHQEcVFAKeR41fkiLdIPrefOvVG1VZ96U0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_gox",
        importpath = "github.com/mitchellh/gox",
        sum = "h1:lfGJxY7ToLJQjHHwi0EX6uYBdK78egf954SQl13PQJc=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_mitchellh_iochan",
        importpath = "github.com/mitchellh/iochan",
        sum = "h1:C+X3KsSTLFVBr/tK1eYN/vs4rJcvsiLU338UhYPJWeY=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_mitchellh_mapstructure",
        importpath = "github.com/mitchellh/mapstructure",
        sum = "h1:CpVNEelQCZBooIPDn+AR3NpivK/TIKU8bDxdASFVQag=",
        version = "v1.4.1",
    )
    go_repository(
        name = "com_github_mitchellh_pointerstructure",
        importpath = "github.com/mitchellh/pointerstructure",
        sum = "h1:O+i9nHnXS3l/9Wu7r4NrEdwA2VFTicjUEN1uBnDo34A=",
        version = "v1.2.0",
    )

    go_repository(
        name = "com_github_modern_go_concurrent",
        importpath = "github.com/modern-go/concurrent",
        sum = "h1:TRLaZ9cD/w8PVh93nsPXa1VrQ6jlwL5oN8l14QlcNfg=",
        version = "v0.0.0-20180306012644-bacd9c7ef1dd",
    )
    go_repository(
        name = "com_github_modern_go_reflect2",
        importpath = "github.com/modern-go/reflect2",
        sum = "h1:9f412s+6RmYXLWZSEzVVgPGK7C2PphHj5RJrvfx9AWI=",
        version = "v1.0.1",
    )

    go_repository(
        name = "com_github_mohae_deepcopy",
        importpath = "github.com/mohae/deepcopy",
        sum = "h1:RWengNIwukTxcDr9M+97sNutRR1RKhG96O6jWumTTnw=",
        version = "v0.0.0-20170929034955-c48cc78d4826",
    )
    go_repository(
        name = "com_github_mr_tron_base58",
        importpath = "github.com/mr-tron/base58",
        sum = "h1:T/HDJBh4ZCPbU39/+c3rRvE0uKBQlU27+QI8LJ4t64o=",
        version = "v1.2.0",
    )

    go_repository(
        name = "com_github_mschoch_smat",
        importpath = "github.com/mschoch/smat",
        sum = "h1:VeRdUYdCw49yizlSbMEn2SZ+gT+3IUKx8BqxyQdz+BY=",
        version = "v0.0.0-20160514031455-90eadee771ae",
    )
    go_repository(
        name = "com_github_multiformats_go_base32",
        importpath = "github.com/multiformats/go-base32",
        sum = "h1:+qMh4a2f37b4xTNs6mqitDinryCI+tfO2dRVMN9mjSE=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_multiformats_go_base36",
        importpath = "github.com/multiformats/go-base36",
        sum = "h1:JR6TyF7JjGd3m6FbLU2cOxhC0Li8z8dLNGQ89tUg4F4=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_multiformats_go_multiaddr",
        importpath = "github.com/multiformats/go-multiaddr",
        sum = "h1:qMnoOPj2s8xxPU5kZ57Cqdr0hHhARz7mFsPMIiYNqzg=",
        version = "v0.6.0",
    )
    go_repository(
        name = "com_github_multiformats_go_multiaddr_dns",
        importpath = "github.com/multiformats/go-multiaddr-dns",
        sum = "h1:QgQgR+LQVt3NPTjbrLLpsaT2ufAA2y0Mkk+QRVJbW3A=",
        version = "v0.3.1",
    )
    go_repository(
        name = "com_github_multiformats_go_multiaddr_fmt",
        importpath = "github.com/multiformats/go-multiaddr-fmt",
        sum = "h1:WLEFClPycPkp4fnIzoFoV9FVd49/eQsuaL3/CWe167E=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_multiformats_go_multiaddr_net",
        importpath = "github.com/multiformats/go-multiaddr-net",
        sum = "h1:MSXRGN0mFymt6B1yo/6BPnIRpLPEnKgQNvVfCX5VDJk=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_multiformats_go_multibase",
        importpath = "github.com/multiformats/go-multibase",
        sum = "h1:3ASCDsuLX8+j4kx58qnJ4YFq/JWTJpCyDW27ztsVTOI=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_multiformats_go_multicodec",
        importpath = "github.com/multiformats/go-multicodec",
        sum = "h1:EgU6cBe/D7WRwQb1KmnBvU7lrcFGMggZVTPtOW9dDHs=",
        version = "v0.5.0",
    )

    go_repository(
        name = "com_github_multiformats_go_multihash",
        importpath = "github.com/multiformats/go-multihash",
        sum = "h1:oytJb9ZA1OUW0r0f9ea18GiaPOo4SXyc7p2movyUuo4=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_multiformats_go_multistream",
        importpath = "github.com/multiformats/go-multistream",
        sum = "h1:d5PZpjwRgVlbwfdTDjife7XszfZd8KYWfROYFlGcR8o=",
        version = "v0.3.3",
    )
    go_repository(
        name = "com_github_multiformats_go_varint",
        importpath = "github.com/multiformats/go-varint",
        sum = "h1:gk85QWKxh3TazbLxED/NlDVv8+q+ReFJk7Y2W/KhfNY=",
        version = "v0.0.6",
    )
    go_repository(
        name = "com_github_munnerz_goautoneg",
        importpath = "github.com/munnerz/goautoneg",
        sum = "h1:7PxY7LVfSZm7PEeBTyK1rj1gABdCO2mbri6GKO1cMDs=",
        version = "v0.0.0-20120707110453-a547fc61f48d",
    )
    go_repository(
        name = "com_github_mwitkow_go_conntrack",
        importpath = "github.com/mwitkow/go-conntrack",
        sum = "h1:KUppIJq7/+SVif2QVs3tOP0zanoHgBEVAwHxUSIzRqU=",
        version = "v0.0.0-20190716064945-2f068394615f",
    )
    go_repository(
        name = "com_github_mxk_go_flowrate",
        importpath = "github.com/mxk/go-flowrate",
        sum = "h1:y5//uYreIhSUg3J1GEMiLbxo1LJaP8RfCpH6pymGZus=",
        version = "v0.0.0-20140419014527-cca7078d478f",
    )
    go_repository(
        name = "com_github_naoina_go_stringutil",
        importpath = "github.com/naoina/go-stringutil",
        sum = "h1:rCUeRUHjBjGTSHl0VC00jUPLz8/F9dDzYI70Hzifhks=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_naoina_toml",
        importpath = "github.com/naoina/toml",
        sum = "h1:shk/vn9oCoOTmwcouEdwIeOtOGA/ELRUw/GwvxwfT+0=",
        version = "v0.1.2-0.20170918210437-9fafd6967416",
    )
    go_repository(
        name = "com_github_nats_io_jwt",
        importpath = "github.com/nats-io/jwt",
        sum = "h1:+RB5hMpXUUA2dfxuhBTEkMOrYmM+gKIZYS1KjSostMI=",
        version = "v0.3.2",
    )
    go_repository(
        name = "com_github_nats_io_nats_go",
        importpath = "github.com/nats-io/nats.go",
        sum = "h1:ik3HbLhZ0YABLto7iX80pZLPw/6dx3T+++MZJwLnMrQ=",
        version = "v1.9.1",
    )
    go_repository(
        name = "com_github_nats_io_nats_server_v2",
        importpath = "github.com/nats-io/nats-server/v2",
        sum = "h1:i2Ly0B+1+rzNZHHWtD4ZwKi+OU5l+uQo1iDHZ2PmiIc=",
        version = "v2.1.2",
    )
    go_repository(
        name = "com_github_nats_io_nkeys",
        importpath = "github.com/nats-io/nkeys",
        sum = "h1:6JrEfig+HzTH85yxzhSVbjHRJv9cn0p6n3IngIcM5/k=",
        version = "v0.1.3",
    )
    go_repository(
        name = "com_github_nats_io_nuid",
        importpath = "github.com/nats-io/nuid",
        sum = "h1:5iA8DT8V7q8WK2EScv2padNa/rTESc1KdnPw4TC2paw=",
        version = "v1.0.1",
    )

    go_repository(
        name = "com_github_neelance_astrewrite",
        importpath = "github.com/neelance/astrewrite",
        sum = "h1:D6paGObi5Wud7xg83MaEFyjxQB1W5bz5d0IFppr+ymk=",
        version = "v0.0.0-20160511093645-99348263ae86",
    )
    go_repository(
        name = "com_github_neelance_sourcemap",
        importpath = "github.com/neelance/sourcemap",
        sum = "h1:eFXv9Nu1lGbrNbj619aWwZfVF5HBrm9Plte8aNptuTI=",
        version = "v0.0.0-20151028013722-8c68805598ab",
    )

    go_repository(
        name = "com_github_nishanths_predeclared",
        importpath = "github.com/nishanths/predeclared",
        sum = "h1:3f0nxAmdj/VoCGN/ijdMy7bj6SBagaqYg1B0hu8clMA=",
        version = "v0.0.0-20200524104333-86fad755b4d3",
    )

    go_repository(
        name = "com_github_nxadm_tail",
        importpath = "github.com/nxadm/tail",
        sum = "h1:nPr65rt6Y5JFSKQO7qToXr7pePgD6Gwiw05lkbyAQTE=",
        version = "v1.4.8",
    )
    go_repository(
        name = "com_github_nytimes_gziphandler",
        importpath = "github.com/NYTimes/gziphandler",
        sum = "h1:lsxEuwrXEAokXB9qhlbKWPpo3KMLZQ5WB5WLQRW1uq0=",
        version = "v0.0.0-20170623195520-56545f4a5d46",
    )
    go_repository(
        name = "com_github_oklog_oklog",
        importpath = "github.com/oklog/oklog",
        sum = "h1:wVfs8F+in6nTBMkA7CbRw+zZMIB7nNM825cM1wuzoTk=",
        version = "v0.3.2",
    )
    go_repository(
        name = "com_github_oklog_run",
        importpath = "github.com/oklog/run",
        sum = "h1:Ru7dDtJNOyC66gQ5dQmaCa0qIsAUFY3sFpK1Xk8igrw=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_oklog_ulid",
        importpath = "github.com/oklog/ulid",
        sum = "h1:EGfNDEx6MqHz8B3uNV6QAib1UR2Lm97sHi3ocA6ESJ4=",
        version = "v1.3.1",
    )
    go_repository(
        name = "com_github_olekukonko_tablewriter",
        importpath = "github.com/olekukonko/tablewriter",
        sum = "h1:P2Ga83D34wi1o9J6Wh1mRuqd4mF/x/lgBS7N7AbDhec=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_oneofone_xxhash",
        importpath = "github.com/OneOfOne/xxhash",
        sum = "h1:KMrpdQIwFcEqXDklaen+P1axHaj9BSKzvpUUfnHldSE=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_onsi_ginkgo",
        importpath = "github.com/onsi/ginkgo",
        sum = "h1:8xi0RTUf59SOSfEtZMvwTvXYMzG4gV23XVHOZiXNtnE=",
        version = "v1.16.5",
    )
    go_repository(
        name = "com_github_onsi_ginkgo_v2",
        importpath = "github.com/onsi/ginkgo/v2",
        sum = "h1:e/3Cwtogj0HA+25nMP1jCMDIf8RtRYbGwGGuBIFztkc=",
        version = "v2.1.3",
    )

    go_repository(
        name = "com_github_onsi_gomega",
        importpath = "github.com/onsi/gomega",
        sum = "h1:4ieX6qQjPP/BfC3mpsAtIGGlxTWPeA3Inl/7DtXw1tw=",
        version = "v1.19.0",
    )
    go_repository(
        name = "com_github_op_go_logging",
        importpath = "github.com/op/go-logging",
        sum = "h1:lDH9UUVJtmYCjyT0CI4q8xvlXPxeZ0gYCVvWbmPlp88=",
        version = "v0.0.0-20160315200505-970db520ece7",
    )

    go_repository(
        name = "com_github_openconfig_gnmi",
        importpath = "github.com/openconfig/gnmi",
        sum = "h1:a380JP+B7xlMbEQOlha1buKhzBPXFqgFXplyWCEIGEY=",
        version = "v0.0.0-20190823184014-89b2bf29312c",
    )
    go_repository(
        name = "com_github_openconfig_reference",
        importpath = "github.com/openconfig/reference",
        sum = "h1:yHCGAHg2zMaW8olLrqEt3SAHGcEx2aJPEQWMRCyravY=",
        version = "v0.0.0-20190727015836-8dfd928c9696",
    )
    go_repository(
        name = "com_github_opencontainers_runtime_spec",
        importpath = "github.com/opencontainers/runtime-spec",
        sum = "h1:UfAcuLBJB9Coz72x1hgl8O5RVzTdNiaglX6v2DM6FI0=",
        version = "v1.0.2",
    )

    go_repository(
        name = "com_github_opentracing_basictracer_go",
        importpath = "github.com/opentracing/basictracer-go",
        sum = "h1:YyUAhaEfjoWXclZVJ9sGoNct7j4TVk7lZWlQw5UXuoo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_opentracing_contrib_go_observer",
        importpath = "github.com/opentracing-contrib/go-observer",
        sum = "h1:lM6RxxfUMrYL/f8bWEUqdXrANWtrL7Nndbm9iFN0DlU=",
        version = "v0.0.0-20170622124052-a52f23424492",
    )

    go_repository(
        name = "com_github_opentracing_opentracing_go",
        importpath = "github.com/opentracing/opentracing-go",
        sum = "h1:uEJPy/1a5RIPAJ0Ov+OIO8OxWu77jEv+1B0VhjKrZUs=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_openzipkin_contrib_zipkin_go_opentracing",
        importpath = "github.com/openzipkin-contrib/zipkin-go-opentracing",
        sum = "h1:ZCnq+JUrvXcDVhX/xRolRBZifmabN1HcS1wrPSvxhrU=",
        version = "v0.4.5",
    )
    go_repository(
        name = "com_github_openzipkin_zipkin_go",
        importpath = "github.com/openzipkin/zipkin-go",
        sum = "h1:nY8Hti+WKaP0cRsSeQ026wU03QsM762XBeCXBb9NAWI=",
        version = "v0.2.2",
    )
    go_repository(
        name = "com_github_otiai10_copy",
        importpath = "github.com/otiai10/copy",
        sum = "h1:HvG945u96iNadPoG2/Ja2+AUJeW5YuFQMixq9yirC+k=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_otiai10_curr",
        importpath = "github.com/otiai10/curr",
        sum = "h1:TJIWdbX0B+kpNagQrjgq8bCMrbhiuX73M2XwgtDMoOI=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_otiai10_mint",
        importpath = "github.com/otiai10/mint",
        sum = "h1:BCmzIS3n71sGfHB5NMNDB3lHYPz8fWSkCAErHed//qc=",
        version = "v1.3.1",
    )

    go_repository(
        name = "com_github_pact_foundation_pact_go",
        importpath = "github.com/pact-foundation/pact-go",
        sum = "h1:OYkFijGHoZAYbOIb1LWXrwKQbMMRUv1oQ89blD2Mh2Q=",
        version = "v1.0.4",
    )
    go_repository(
        name = "com_github_pascaldekloe_goe",
        importpath = "github.com/pascaldekloe/goe",
        sum = "h1:Lgl0gzECD8GnQ5QCWA8o6BtfL6mDH5rQgM4/fX3avOs=",
        version = "v0.0.0-20180627143212-57f6aae5913c",
    )

    go_repository(
        name = "com_github_patrickmn_go_cache",
        importpath = "github.com/patrickmn/go-cache",
        sum = "h1:HRMgzkcYKYpi3C8ajMPV8OFXaaRUnok+kx1WdO15EQc=",
        version = "v2.1.0+incompatible",
    )
    go_repository(
        name = "com_github_paulbellamy_ratecounter",
        importpath = "github.com/paulbellamy/ratecounter",
        sum = "h1:2L/RhJq+HA8gBQImDXtLPrDXK5qAj6ozWVK/zFXVJGs=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_pbnjay_memory",
        importpath = "github.com/pbnjay/memory",
        sum = "h1:onHthvaw9LFnH4t2DcNVpwGmV9E1BkGknEliJkfwQj0=",
        version = "v0.0.0-20210728143218-7b4eea64cf58",
    )

    go_repository(
        name = "com_github_pborman_uuid",
        importpath = "github.com/pborman/uuid",
        sum = "h1:+ZZIw58t/ozdjRaXh/3awHfmWRbzYxJoAdNJxe/3pvw=",
        version = "v1.2.1",
    )
    go_repository(
        name = "com_github_pelletier_go_toml",
        importpath = "github.com/pelletier/go-toml",
        sum = "h1:T5zMGML61Wp+FlcbWjRDT7yAxhJNAiPPLOFECq181zc=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_performancecopilot_speed",
        importpath = "github.com/performancecopilot/speed",
        sum = "h1:2WnRzIquHa5QxaJKShDkLM+sc0JPuwhXzK8OYOyt3Vg=",
        version = "v3.0.0+incompatible",
    )

    go_repository(
        name = "com_github_peterbourgon_diskv",
        importpath = "github.com/peterbourgon/diskv",
        sum = "h1:UBdAOUP5p4RWqPBg048CAvpKN+vxiaj6gdUUzhl4XmI=",
        version = "v2.0.1+incompatible",
    )

    go_repository(
        name = "com_github_peterh_liner",
        importpath = "github.com/peterh/liner",
        sum = "h1:w/UPXyl5GfahFxcTOz2j9wCIHNI+pUPr2laqpojKNCg=",
        version = "v1.2.0",
    )

    go_repository(
        name = "com_github_philhofer_fwd",
        importpath = "github.com/philhofer/fwd",
        sum = "h1:UbZqGr5Y38ApvM/V/jEljVxwocdweyH+vmYvRPBnbqQ=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_pierrec_lz4",
        importpath = "github.com/pierrec/lz4",
        sum = "h1:mFe7ttWaflA46Mhqh+jUfjp2qTbPYxLB2/OyBppH9dg=",
        version = "v2.4.1+incompatible",
    )
    go_repository(
        name = "com_github_pkg_diff",
        importpath = "github.com/pkg/diff",
        sum = "h1:aoZm08cpOy4WuID//EZDgcC4zIxODThtZNPirFr42+A=",
        version = "v0.0.0-20210226163009-20ebb0f2a09e",
    )

    go_repository(
        name = "com_github_pkg_errors",
        importpath = "github.com/pkg/errors",
        sum = "h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt4=",
        version = "v0.9.1",
    )
    go_repository(
        name = "com_github_pkg_profile",
        importpath = "github.com/pkg/profile",
        sum = "h1:042Buzk+NhDI+DeSAA62RwJL8VAuZUMQZUjCsRz1Mug=",
        version = "v1.5.0",
    )

    go_repository(
        name = "com_github_pkg_term",
        importpath = "github.com/pkg/term",
        sum = "h1:tFwafIEMf0B7NlcxV/zJ6leBIa81D3hgGSgsE5hCkOQ=",
        version = "v0.0.0-20180730021639-bffc007b7fd5",
    )
    go_repository(
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_posener_complete",
        importpath = "github.com/posener/complete",
        sum = "h1:ccV59UEOTzVDnDUEFdT95ZzHVZ+5+158q8+SJb2QV5w=",
        version = "v1.1.1",
    )

    go_repository(
        name = "com_github_prometheus_client_golang",
        importpath = "github.com/prometheus/client_golang",
        sum = "h1:51L9cDoUHVrXx4zWYlcLQIZ+d+VXHgqnYKkIuq4g/34=",
        version = "v1.12.2",
    )
    go_repository(
        name = "com_github_prometheus_client_model",
        importpath = "github.com/prometheus/client_model",
        sum = "h1:uq5h0d+GuxiXLJLNABMgp2qUWDPiLvgCzz2dUR+/W/M=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_prometheus_common",
        importpath = "github.com/prometheus/common",
        sum = "h1:Eyr+Pw2VymWejHqCugNaQXkAi6KayVNxaHeu6khmFBE=",
        version = "v0.35.0",
    )
    go_repository(
        name = "com_github_prometheus_procfs",
        importpath = "github.com/prometheus/procfs",
        sum = "h1:4jVXhlkAyzOScmCkXBTOLRLTz8EeU+eyjrwB/EPq0VU=",
        version = "v0.7.3",
    )
    go_repository(
        name = "com_github_prometheus_prom2json",
        importpath = "github.com/prometheus/prom2json",
        sum = "h1:BlqrtbT9lLH3ZsOVhXPsHzFrApCTKRifB7gjJuypu6Y=",
        version = "v1.3.0",
    )

    go_repository(
        name = "com_github_prometheus_tsdb",
        importpath = "github.com/prometheus/tsdb",
        sum = "h1:If5rVCMTp6W2SiRAQFlbpJNgVlgMEd+U2GZckwK38ic=",
        version = "v0.10.0",
    )
    go_repository(
        name = "com_github_prysmaticlabs_fastssz",
        importpath = "github.com/prysmaticlabs/fastssz",
        sum = "h1:Y3PcvUrnneMWLuypZpwPz8P70/DQsz6KgV9JveKpyZs=",
        version = "v0.0.0-20220628121656-93dfe28febab",
    )

    go_repository(
        name = "com_github_prysmaticlabs_go_bitfield",
        importpath = "github.com/prysmaticlabs/go-bitfield",
        sum = "h1:0tVE4tdWQK9ZpYygoV7+vS6QkDvQVySboMVEIxBJmXw=",
        version = "v0.0.0-20210809151128-385d8c5e3fb7",
    )
    go_repository(
        name = "com_github_prysmaticlabs_gohashtree",
        importpath = "github.com/prysmaticlabs/gohashtree",
        sum = "h1:hk5ZsDQuSkyUMhTd55qB396P1+dtyIKiSwMmYE/hyEU=",
        version = "v0.0.2-alpha",
    )

    go_repository(
        name = "com_github_prysmaticlabs_prombbolt",
        importpath = "github.com/prysmaticlabs/prombbolt",
        sum = "h1:9PHRCuO/VN0s9k+RmLykho7AjDxblNYI5bYKed16NPU=",
        version = "v0.0.0-20210126082820-9b7adba6db7c",
    )
    go_repository(
        name = "com_github_prysmaticlabs_protoc_gen_go_cast",
        importpath = "github.com/prysmaticlabs/protoc-gen-go-cast",
        sum = "h1:+jhXLjEYVW4qU2z5SOxlxN+Hv/A9FDf0HpfDurfMEz0=",
        version = "v0.0.0-20211014160335-757fae4f38c6",
    )

    go_repository(
        name = "com_github_puerkitobio_purell",
        importpath = "github.com/PuerkitoBio/purell",
        sum = "h1:0GoNN3taZV6QI81IXgCbxMyEaJDXMSIjArYBCYzVVvs=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_puerkitobio_urlesc",
        importpath = "github.com/PuerkitoBio/urlesc",
        sum = "h1:JCHLVE3B+kJde7bIEo5N4J+ZbLhp0J1Fs+ulyRws4gE=",
        version = "v0.0.0-20160726150825-5bd2802263f2",
    )
    go_repository(
        name = "com_github_r3labs_sse",
        importpath = "github.com/r3labs/sse",
        sum = "h1:zAsgcP8MhzAbhMnB1QQ2O7ZhWYVGYSR2iVcjzQuPV+o=",
        version = "v0.0.0-20210224172625-26fe804710bc",
    )
    go_repository(
        name = "com_github_raulk_clock",
        importpath = "github.com/raulk/clock",
        sum = "h1:dpb29+UKMbLqiU/jqIJptgLR1nn23HLgMY0sTCDza5Y=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_raulk_go_watchdog",
        importpath = "github.com/raulk/go-watchdog",
        sum = "h1:konN75pw2BMmZ+AfuAm5rtFsWcJpKF3m02rKituuXNo=",
        version = "v1.2.0",
    )

    go_repository(
        name = "com_github_rcrowley_go_metrics",
        importpath = "github.com/rcrowley/go-metrics",
        sum = "h1:dY6ETXrvDG7Sa4vE8ZQG4yqWg6UnOcbqTAahkV813vQ=",
        version = "v0.0.0-20190826022208-cac0b30c2563",
    )

    go_repository(
        name = "com_github_retailnext_hllpp",
        importpath = "github.com/retailnext/hllpp",
        sum = "h1:RnWNS9Hlm8BIkjr6wx8li5abe0fr73jljLycdfemTp0=",
        version = "v1.0.1-0.20180308014038-101a6d2f8b52",
    )
    go_repository(
        name = "com_github_rivo_tview",
        importpath = "github.com/rivo/tview",
        sum = "h1:rqaqSUdaW+OBbjnsrOoiaJv43mSRARuvsAuirmdxu7E=",
        version = "v0.0.0-20200712113419-c65badfc3d92",
    )
    go_repository(
        name = "com_github_rivo_uniseg",
        importpath = "github.com/rivo/uniseg",
        sum = "h1:+2KBaVoUmb9XzDsrx/Ct0W/EYOSFf/nWTauy++DprtY=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_rjeczalik_notify",
        importpath = "github.com/rjeczalik/notify",
        sum = "h1:CLCKso/QK1snAlnhNR/CNvNiFU2saUtjV0bx3EwNeCE=",
        version = "v0.9.1",
    )

    go_repository(
        name = "com_github_rogpeppe_fastuuid",
        importpath = "github.com/rogpeppe/fastuuid",
        sum = "h1:Ppwyp6VYCF1nvBTXL3trRso7mXMlRrw9ooo375wvi2s=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_rogpeppe_go_internal",
        importpath = "github.com/rogpeppe/go-internal",
        sum = "h1:FCbCCtXNOY3UtUuHUYaghJg4y7Fd14rXifAYUAtL9R8=",
        version = "v1.8.0",
    )

    go_repository(
        name = "com_github_rs_cors",
        importpath = "github.com/rs/cors",
        sum = "h1:+88SsELBHx5r+hZ8TCkggzSstaWNbDvThkVK8H6f9ik=",
        version = "v1.7.0",
    )
    go_repository(
        name = "com_github_rs_xhandler",
        importpath = "github.com/rs/xhandler",
        sum = "h1:3hxavr+IHMsQBrYUPQM5v0CgENFktkkbg1sfpgM3h20=",
        version = "v0.0.0-20160618193221-ed27b6fd6521",
    )

    go_repository(
        name = "com_github_russross_blackfriday",
        importpath = "github.com/russross/blackfriday",
        sum = "h1:HyvC0ARfnZBqnXwABFeSZHpKvJHJJfPz81GNueLj0oo=",
        version = "v1.5.2",
    )

    go_repository(
        name = "com_github_russross_blackfriday_v2",
        importpath = "github.com/russross/blackfriday/v2",
        sum = "h1:JIOH55/0cWyOuilr9/qlrm0BSXldqnqwMsf35Ld67mk=",
        version = "v2.1.0",
    )
    go_repository(
        name = "com_github_ryanuber_columnize",
        importpath = "github.com/ryanuber/columnize",
        sum = "h1:UFr9zpz4xgTnIE5yIMtWAMngCdZ9p/+q6lTbgelo80M=",
        version = "v0.0.0-20160712163229-9b3edd62028f",
    )
    go_repository(
        name = "com_github_samuel_go_zookeeper",
        importpath = "github.com/samuel/go-zookeeper",
        sum = "h1:p3Vo3i64TCLY7gIfzeQaUJ+kppEO5WQG3cL8iE8tGHU=",
        version = "v0.0.0-20190923202752-2cc03de413da",
    )

    go_repository(
        name = "com_github_satori_go_uuid",
        importpath = "github.com/satori/go.uuid",
        sum = "h1:0uYX9dsZ2yD7q2RtLRtPSdGDWzjeM3TbMJP9utgA0ww=",
        version = "v1.2.0",
    )

    go_repository(
        name = "com_github_schollz_progressbar_v3",
        importpath = "github.com/schollz/progressbar/v3",
        sum = "h1:nMinx+JaEm/zJz4cEyClQeAw5rsYSB5th3xv+5lV6Vg=",
        version = "v3.3.4",
    )
    go_repository(
        name = "com_github_sean_seed",
        importpath = "github.com/sean-/seed",
        sum = "h1:nn5Wsu0esKSJiIVhscUtVbo7ada43DJhG55ua/hjS5I=",
        version = "v0.0.0-20170313163322-e2103e2c3529",
    )
    go_repository(
        name = "com_github_segmentio_kafka_go",
        importpath = "github.com/segmentio/kafka-go",
        sum = "h1:HtCSf6B4gN/87yc5qTl7WsxPKQIIGXLPPM1bMCPOsoY=",
        version = "v0.2.0",
    )

    go_repository(
        name = "com_github_sergi_go_diff",
        importpath = "github.com/sergi/go-diff",
        sum = "h1:Kpca3qRNrduNnOQeazBd0ysaKrUJiIuISHxogkT9RPQ=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_shirou_gopsutil",
        importpath = "github.com/shirou/gopsutil",
        sum = "h1:Bn1aCHHRnjv4Bl16T8rcaFjYSrGrIZvpiGO6P3Q4GpU=",
        version = "v3.21.4-0.20210419000835-c7a38de76ee5+incompatible",
    )
    go_repository(
        name = "com_github_shopify_sarama",
        importpath = "github.com/Shopify/sarama",
        sum = "h1:3jnfWKD7gVwbB1KSy/lE0szA9duPuSFLViK0o/d3DgA=",
        version = "v1.26.1",
    )
    go_repository(
        name = "com_github_shopify_toxiproxy",
        importpath = "github.com/Shopify/toxiproxy",
        sum = "h1:TKdv8HiTLgE5wdJuEML90aBgNWsokNbMijUGhmcoBJc=",
        version = "v2.1.4+incompatible",
    )
    go_repository(
        name = "com_github_shurcool_component",
        importpath = "github.com/shurcooL/component",
        sum = "h1:Fth6mevc5rX7glNLpbAMJnqKlfIkcTjZCSHEeqvKbcI=",
        version = "v0.0.0-20170202220835-f88ec8f54cc4",
    )
    go_repository(
        name = "com_github_shurcool_events",
        importpath = "github.com/shurcooL/events",
        sum = "h1:vabduItPAIz9px5iryD5peyx7O3Ya8TBThapgXim98o=",
        version = "v0.0.0-20181021180414-410e4ca65f48",
    )
    go_repository(
        name = "com_github_shurcool_github_flavored_markdown",
        importpath = "github.com/shurcooL/github_flavored_markdown",
        sum = "h1:qb9IthCFBmROJ6YBS31BEMeSYjOscSiG+EO+JVNTz64=",
        version = "v0.0.0-20181002035957-2122de532470",
    )
    go_repository(
        name = "com_github_shurcool_go",
        importpath = "github.com/shurcooL/go",
        sum = "h1:MZM7FHLqUHYI0Y/mQAt3d2aYa0SiNms/hFqC9qJYolM=",
        version = "v0.0.0-20180423040247-9e1955d9fb6e",
    )
    go_repository(
        name = "com_github_shurcool_go_goon",
        importpath = "github.com/shurcooL/go-goon",
        sum = "h1:llrF3Fs4018ePo4+G/HV/uQUqEI1HMDjCeOf2V6puPc=",
        version = "v0.0.0-20170922171312-37c2f522c041",
    )
    go_repository(
        name = "com_github_shurcool_gofontwoff",
        importpath = "github.com/shurcooL/gofontwoff",
        sum = "h1:Yoy/IzG4lULT6qZg62sVC+qyBL8DQkmD2zv6i7OImrc=",
        version = "v0.0.0-20180329035133-29b52fc0a18d",
    )
    go_repository(
        name = "com_github_shurcool_gopherjslib",
        importpath = "github.com/shurcooL/gopherjslib",
        sum = "h1:UOk+nlt1BJtTcH15CT7iNO7YVWTfTv/DNwEAQHLIaDQ=",
        version = "v0.0.0-20160914041154-feb6d3990c2c",
    )
    go_repository(
        name = "com_github_shurcool_highlight_diff",
        importpath = "github.com/shurcooL/highlight_diff",
        sum = "h1:vYEG87HxbU6dXj5npkeulCS96Dtz5xg3jcfCgpcvbIw=",
        version = "v0.0.0-20170515013008-09bb4053de1b",
    )
    go_repository(
        name = "com_github_shurcool_highlight_go",
        importpath = "github.com/shurcooL/highlight_go",
        sum = "h1:7pDq9pAMCQgRohFmd25X8hIH8VxmT3TaDm+r9LHxgBk=",
        version = "v0.0.0-20181028180052-98c3abbbae20",
    )
    go_repository(
        name = "com_github_shurcool_home",
        importpath = "github.com/shurcooL/home",
        sum = "h1:MPblCbqA5+z6XARjScMfz1TqtJC7TuTRj0U9VqIBs6k=",
        version = "v0.0.0-20181020052607-80b7ffcb30f9",
    )
    go_repository(
        name = "com_github_shurcool_htmlg",
        importpath = "github.com/shurcooL/htmlg",
        sum = "h1:crYRwvwjdVh1biHzzciFHe8DrZcYrVcZFlJtykhRctg=",
        version = "v0.0.0-20170918183704-d01228ac9e50",
    )
    go_repository(
        name = "com_github_shurcool_httperror",
        importpath = "github.com/shurcooL/httperror",
        sum = "h1:eHRtZoIi6n9Wo1uR+RU44C247msLWwyA89hVKwRLkMk=",
        version = "v0.0.0-20170206035902-86b7830d14cc",
    )
    go_repository(
        name = "com_github_shurcool_httpfs",
        importpath = "github.com/shurcooL/httpfs",
        sum = "h1:SWV2fHctRpRrp49VXJ6UZja7gU9QLHwRpIPBN89SKEo=",
        version = "v0.0.0-20171119174359-809beceb2371",
    )
    go_repository(
        name = "com_github_shurcool_httpgzip",
        importpath = "github.com/shurcooL/httpgzip",
        sum = "h1:fxoFD0in0/CBzXoyNhMTjvBZYW6ilSnTw7N7y/8vkmM=",
        version = "v0.0.0-20180522190206-b1c53ac65af9",
    )
    go_repository(
        name = "com_github_shurcool_issues",
        importpath = "github.com/shurcooL/issues",
        sum = "h1:T4wuULTrzCKMFlg3HmKHgXAF8oStFb/+lOIupLV2v+o=",
        version = "v0.0.0-20181008053335-6292fdc1e191",
    )
    go_repository(
        name = "com_github_shurcool_issuesapp",
        importpath = "github.com/shurcooL/issuesapp",
        sum = "h1:Y+TeIabU8sJD10Qwd/zMty2/LEaT9GNDaA6nyZf+jgo=",
        version = "v0.0.0-20180602232740-048589ce2241",
    )
    go_repository(
        name = "com_github_shurcool_notifications",
        importpath = "github.com/shurcooL/notifications",
        sum = "h1:TQVQrsyNaimGwF7bIhzoVC9QkKm4KsWd8cECGzFx8gI=",
        version = "v0.0.0-20181007000457-627ab5aea122",
    )
    go_repository(
        name = "com_github_shurcool_octicon",
        importpath = "github.com/shurcooL/octicon",
        sum = "h1:bu666BQci+y4S0tVRVjsHUeRon6vUXmsGBwdowgMrg4=",
        version = "v0.0.0-20181028054416-fa4f57f9efb2",
    )
    go_repository(
        name = "com_github_shurcool_reactions",
        importpath = "github.com/shurcooL/reactions",
        sum = "h1:LneqU9PHDsg/AkPDU3AkqMxnMYL+imaqkpflHu73us8=",
        version = "v0.0.0-20181006231557-f2e0b4ca5b82",
    )

    go_repository(
        name = "com_github_shurcool_sanitized_anchor_name",
        importpath = "github.com/shurcooL/sanitized_anchor_name",
        sum = "h1:PdmoCO6wvbs+7yrJyMORt4/BmY5IYyJwS/kOiWx8mHo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_shurcool_users",
        importpath = "github.com/shurcooL/users",
        sum = "h1:YGaxtkYjb8mnTvtufv2LKLwCQu2/C7qFB7UtrOlTWOY=",
        version = "v0.0.0-20180125191416-49c67e49c537",
    )
    go_repository(
        name = "com_github_shurcool_webdavfs",
        importpath = "github.com/shurcooL/webdavfs",
        sum = "h1:JtcyT0rk/9PKOdnKQzuDR+FSjh7SGtJwpgVpfZBRKlQ=",
        version = "v0.0.0-20170829043945-18c3829fa133",
    )

    go_repository(
        name = "com_github_sirupsen_logrus",
        importpath = "github.com/sirupsen/logrus",
        sum = "h1:dJKuHgqk1NNQlqoA6BTlM1Wf9DOH3NBjQyu0h9+AZZE=",
        version = "v1.8.1",
    )

    go_repository(
        name = "com_github_smartystreets_assertions",
        importpath = "github.com/smartystreets/assertions",
        sum = "h1:zE9ykElWQ6/NYmHa3jpm/yHnI4xSofP+UP6SpjHcSeM=",
        version = "v0.0.0-20180927180507-b2de0cb4f26d",
    )
    go_repository(
        name = "com_github_smartystreets_goconvey",
        importpath = "github.com/smartystreets/goconvey",
        sum = "h1:fv0U8FUIMPNf1L9lnHLvLhgicrIVChEkdzIKYqbNC9s=",
        version = "v1.6.4",
    )
    go_repository(
        name = "com_github_smola_gocompat",
        importpath = "github.com/smola/gocompat",
        sum = "h1:6b1oIMlUXIpz//VKEDzPVBK8KG7beVwmHIUEBIs/Pns=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_soheilhy_cmux",
        importpath = "github.com/soheilhy/cmux",
        sum = "h1:0HKaf1o97UwFjHH9o5XsHUOF+tqmdA7KEzXLpiyaw0E=",
        version = "v0.1.4",
    )
    go_repository(
        name = "com_github_sony_gobreaker",
        importpath = "github.com/sony/gobreaker",
        sum = "h1:oMnRNZXX5j85zso6xCPRNPtmAycat+WcoKbklScLDgQ=",
        version = "v0.4.1",
    )
    go_repository(
        name = "com_github_sourcegraph_annotate",
        importpath = "github.com/sourcegraph/annotate",
        sum = "h1:yKm7XZV6j9Ev6lojP2XaIshpT4ymkqhMeSghO5Ps00E=",
        version = "v0.0.0-20160123013949-f4cad6c6324d",
    )
    go_repository(
        name = "com_github_sourcegraph_syntaxhighlight",
        importpath = "github.com/sourcegraph/syntaxhighlight",
        sum = "h1:qpG93cPwA5f7s/ZPBJnGOYQNK/vKsaDaseuKT5Asee8=",
        version = "v0.0.0-20170531221838-bd320f5d308e",
    )

    go_repository(
        name = "com_github_spacemonkeygo_openssl",
        importpath = "github.com/spacemonkeygo/openssl",
        sum = "h1:/eS3yfGjQKG+9kayBkj0ip1BGhq6zJ3eaVksphxAaek=",
        version = "v0.0.0-20181017203307-c2dcc5cca94a",
    )
    go_repository(
        name = "com_github_spacemonkeygo_spacelog",
        importpath = "github.com/spacemonkeygo/spacelog",
        sum = "h1:RC6RW7j+1+HkWaX/Yh71Ee5ZHaHYt7ZP4sQgUrm6cDU=",
        version = "v0.0.0-20180420211403-2296661a0572",
    )

    go_repository(
        name = "com_github_spaolacci_murmur3",
        importpath = "github.com/spaolacci/murmur3",
        sum = "h1:7c1g84S4BPRrfL5Xrdp6fOJ206sU9y293DDHaoy0bLI=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_spf13_afero",
        importpath = "github.com/spf13/afero",
        sum = "h1:5jhuqJyZCZf2JRofRvN/nIFgIWNzPa3/Vz8mYylgbWc=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_spf13_cast",
        importpath = "github.com/spf13/cast",
        sum = "h1:oget//CVOEoFewqQxwr0Ej5yjygnqGkvggSE/gB35Q8=",
        version = "v1.3.0",
    )

    go_repository(
        name = "com_github_spf13_cobra",
        importpath = "github.com/spf13/cobra",
        sum = "h1:KfztREH0tPxJJ+geloSLaAkaPkr4ki2Er5quFV1TDo4=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_spf13_jwalterweatherman",
        importpath = "github.com/spf13/jwalterweatherman",
        sum = "h1:XHEdyB+EcvlqZamSM4ZOMGlc93t6AcsBEu9Gc1vn7yk=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_spf13_pflag",
        importpath = "github.com/spf13/pflag",
        sum = "h1:iy+VFUOCP1a+8yFto/drg2CJ5u0yRoB7fZw3DKv/JXA=",
        version = "v1.0.5",
    )
    go_repository(
        name = "com_github_spf13_viper",
        importpath = "github.com/spf13/viper",
        sum = "h1:xVKxvI7ouOI5I+U9s2eeiUfMaWBVoXA3AWskkrqK0VM=",
        version = "v1.7.0",
    )
    go_repository(
        name = "com_github_src_d_envconfig",
        importpath = "github.com/src-d/envconfig",
        sum = "h1:/AJi6DtjFhZKNx3OB2qMsq7y4yT5//AeSZIe7rk+PX8=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_stackexchange_wmi",
        importpath = "github.com/StackExchange/wmi",
        sum = "h1:5sXbqlSomvdjlRbWyNqkPsJ3Fg+tQZCbgeX1VGljbQY=",
        version = "v0.0.0-20210224194228-fe8f1750fd46",
    )
    go_repository(
        name = "com_github_status_im_keycard_go",
        importpath = "github.com/status-im/keycard-go",
        sum = "h1:Oo2KZNP70KE0+IUJSidPj/BFS/RXNHmKIJOdckzml2E=",
        version = "v0.0.0-20200402102358-957c09536969",
    )
    go_repository(
        name = "com_github_steakknife_bloomfilter",
        importpath = "github.com/steakknife/bloomfilter",
        sum = "h1:gIlAHnH1vJb5vwEjIp5kBj/eu99p/bl0Ay2goiPe5xE=",
        version = "v0.0.0-20180922174646-6819c0d2a570",
    )
    go_repository(
        name = "com_github_steakknife_hamming",
        importpath = "github.com/steakknife/hamming",
        sum = "h1:njlZPzLwU639dk2kqnCPPv+wNjq7Xb6EfUxe/oX0/NM=",
        version = "v0.0.0-20180906055917-c99c65617cd3",
    )

    go_repository(
        name = "com_github_streadway_amqp",
        importpath = "github.com/streadway/amqp",
        sum = "h1:WhxRHzgeVGETMlmVfqhRn8RIeeNoPr2Czh33I4Zdccw=",
        version = "v0.0.0-20190827072141-edfb9018d271",
    )
    go_repository(
        name = "com_github_streadway_handy",
        importpath = "github.com/streadway/handy",
        sum = "h1:AhmOdSHeswKHBjhsLs/7+1voOxT+LLrSk/Nxvk35fug=",
        version = "v0.0.0-20190108123426-d5acb3125c2a",
    )

    go_repository(
        name = "com_github_stretchr_objx",
        importpath = "github.com/stretchr/objx",
        sum = "h1:2vfRuCMp5sSVIDSqO8oNnWJq7mPa6KVP3iPIwFBuy8A=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:4jaiDzPyXQvSd7D0EjG45355tLlV3VOECpq10pLC+8s=",
        version = "v1.7.2",
    )

    go_repository(
        name = "com_github_subosito_gotenv",
        importpath = "github.com/subosito/gotenv",
        sum = "h1:Slr1R9HxAlEKefgq5jn9U+DnETlIUa6HfgEzj0g5d7s=",
        version = "v1.2.0",
    )
    http_archive(
        name = "com_github_supranational_blst",
        urls = [
            "https://github.com/supranational/blst/archive/61758ce4e1d18e6929658ac3a29fa39ad91cd294.tar.gz",
        ],
        strip_prefix = "blst-61758ce4e1d18e6929658ac3a29fa39ad91cd294",
        build_file = "//third_party:blst/blst.BUILD",
        sha256 = "acc022ddcf6181f8f402365d6382ea66c4352a10fe5490defd53b0db89739c42",
    )
    go_repository(
        name = "com_github_syndtr_goleveldb",
        importpath = "github.com/syndtr/goleveldb",
        sum = "h1:1ur3QoCqvE5fl+nylMaIr9PVV1w343YRDtsy+Rwu7XI=",
        version = "v1.0.1-0.20220614013038-64ee5596c38a",
    )
    go_repository(
        name = "com_github_tarm_serial",
        importpath = "github.com/tarm/serial",
        sum = "h1:UyzmZLoiDWMRywV4DUYb9Fbt8uiOSooupjTq10vpvnU=",
        version = "v0.0.0-20180830185346-98f6abe2eb07",
    )

    go_repository(
        name = "com_github_templexxx_cpufeat",
        importpath = "github.com/templexxx/cpufeat",
        sum = "h1:89CEmDvlq/F7SJEOqkIdNDGJXrQIhuIx9D2DBXjavSU=",
        version = "v0.0.0-20180724012125-cef66df7f161",
    )
    go_repository(
        name = "com_github_templexxx_xor",
        importpath = "github.com/templexxx/xor",
        sum = "h1:fj5tQ8acgNUr6O8LEplsxDhUIe2573iLkJc+PqnzZTI=",
        version = "v0.0.0-20191217153810-f85b25db303b",
    )
    go_repository(
        name = "com_github_tenntenn_modver",
        importpath = "github.com/tenntenn/modver",
        sum = "h1:2klLppGhDgzJrScMpkj9Ujy3rXPUspSjAcev9tSEBgA=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_tenntenn_text_transform",
        importpath = "github.com/tenntenn/text/transform",
        sum = "h1:f+jULpRQGxTSkNYKJ51yaw6ChIqO+Je8UqsTKN/cDag=",
        version = "v0.0.0-20200319021203-7eef512accb3",
    )

    go_repository(
        name = "com_github_thomaso_mirodin_intmath",
        importpath = "github.com/thomaso-mirodin/intmath",
        sum = "h1:cR8/SYRgyQCt5cNCMniB/ZScMkhI9nk8U5C7SbISXjo=",
        version = "v0.0.0-20160323211736-5dc6d854e46e",
    )

    go_repository(
        name = "com_github_tinylib_msgp",
        importpath = "github.com/tinylib/msgp",
        sum = "h1:DfdQrzQa7Yh2es9SuLkixqxuXS2SxsdYn0KbdrOGWD8=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_tjfoc_gmsm",
        importpath = "github.com/tjfoc/gmsm",
        sum = "h1:i7c6Za/IlgBvnGxYpfD7L3TGuaS+v6oGcgq+J9/ecEA=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_tklauser_go_sysconf",
        importpath = "github.com/tklauser/go-sysconf",
        sum = "h1:uu3Xl4nkLzQfXNsWn15rPc/HQCJKObbt1dKJeWp3vU4=",
        version = "v0.3.5",
    )
    go_repository(
        name = "com_github_tklauser_numcpus",
        importpath = "github.com/tklauser/numcpus",
        sum = "h1:oyhllyrScuYI6g+h/zUvNXNp1wy7x8qQy3t/piefldA=",
        version = "v0.2.2",
    )

    go_repository(
        name = "com_github_tmc_grpc_websocket_proxy",
        importpath = "github.com/tmc/grpc-websocket-proxy",
        sum = "h1:LnC5Kc/wtumK+WB441p7ynQJzVuNRJiqddSIE3IlSEQ=",
        version = "v0.0.0-20190109142713-0ad062ec5ee5",
    )

    go_repository(
        name = "com_github_trailofbits_go_mutexasserts",
        importpath = "github.com/trailofbits/go-mutexasserts",
        sum = "h1:8LRP+2JK8piIUU16ZDgWDXwjJcuJNTtCzadjTZj8Jf0=",
        version = "v0.0.0-20200708152505-19999e7d3cef",
    )
    go_repository(
        name = "com_github_twitchtv_twirp",
        importpath = "github.com/twitchtv/twirp",
        sum = "h1:3fNSDoSPyq+fTrifIvGue9XM/tptzuhiGY83rxPVNUg=",
        version = "v7.1.0+incompatible",
    )
    go_repository(
        name = "com_github_tyler_smith_go_bip39",
        importpath = "github.com/tyler-smith/go-bip39",
        sum = "h1:5eUemwrMargf3BSLRRCalXT93Ns6pQJIjYQN2nyfOP8=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_uber_jaeger_client_go",
        importpath = "github.com/uber/jaeger-client-go",
        sum = "h1:IxcNZ7WRY1Y3G4poYlx24szfsn/3LvK9QHCq9oQw8+U=",
        version = "v2.25.0+incompatible",
    )
    go_repository(
        name = "com_github_ugorji_go_codec",
        importpath = "github.com/ugorji/go/codec",
        sum = "h1:3SVOIvH7Ae1KRYyQWRjXWJEA9sS/c/pjvH++55Gr648=",
        version = "v0.0.0-20181204163529-d75b2dcb6bc8",
    )

    go_repository(
        name = "com_github_urfave_cli",
        importpath = "github.com/urfave/cli",
        sum = "h1:gsqYFH8bb9ekPA12kRo0hfjngWQjkJPlN9R0N78BoUo=",
        version = "v1.22.2",
    )
    go_repository(
        name = "com_github_urfave_cli_v2",
        importpath = "github.com/urfave/cli/v2",
        sum = "h1:x3p8awjp/2arX+Nl/G2040AZpOCHS/eMJJ1/a+mye4Y=",
        version = "v2.10.2",
    )
    go_repository(
        name = "com_github_uudashr_gocognit",
        importpath = "github.com/uudashr/gocognit",
        sum = "h1:rrSex7oHr3/pPLQ0xoWq108XMU8s678FJcQ+aSfOHa4=",
        version = "v1.0.5",
    )

    go_repository(
        name = "com_github_valyala_bytebufferpool",
        importpath = "github.com/valyala/bytebufferpool",
        sum = "h1:GqA5TC/0021Y/b9FG4Oi9Mr3q7XYx6KllzawFIhcdPw=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_valyala_fasttemplate",
        importpath = "github.com/valyala/fasttemplate",
        sum = "h1:TVEnxayobAdVkhQfrfes2IzOB6o+z4roRkPF52WA1u4=",
        version = "v1.2.1",
    )

    go_repository(
        name = "com_github_viant_assertly",
        importpath = "github.com/viant/assertly",
        sum = "h1:5x1GzBaRteIwTr5RAGFVG14uNeRFxVNbXPWrK2qAgpc=",
        version = "v0.4.8",
    )
    go_repository(
        name = "com_github_viant_toolbox",
        importpath = "github.com/viant/toolbox",
        sum = "h1:6TteTDQ68CjgcCe8wH3D3ZhUQQOJXMTbj/D9rkk2a1k=",
        version = "v0.24.0",
    )

    go_repository(
        name = "com_github_victoriametrics_fastcache",
        importpath = "github.com/VictoriaMetrics/fastcache",
        sum = "h1:C/3Oi3EiBCqufydp1neRZkqcwmEiuRT9c3fqvvgKm5o=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_vividcortex_gohistogram",
        importpath = "github.com/VividCortex/gohistogram",
        sum = "h1:6+hBz+qvs0JOrrNhhmR7lFxo5sINxBCGXrdtl/UvroE=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_wealdtech_go_bytesutil",
        importpath = "github.com/wealdtech/go-bytesutil",
        sum = "h1:ocEg3Ke2GkZ4vQw5lp46rmO+pfqCCTgq35gqOy8JKVc=",
        version = "v1.1.1",
    )

    go_repository(
        name = "com_github_wealdtech_go_eth2_types_v2",
        build_directives = [
            "gazelle:resolve go github.com/herumi/bls-eth-go-binary/bls @herumi_bls_eth_go_binary//:go_default_library",
        ],
        importpath = "github.com/wealdtech/go-eth2-types/v2",
        sum = "h1:tiA6T88M6XQIbrV5Zz53l1G5HtRERcxQfmET225V4Ls=",
        version = "v2.5.2",
    )
    go_repository(
        name = "com_github_wealdtech_go_eth2_util",
        importpath = "github.com/wealdtech/go-eth2-util",
        sum = "h1:2INPeOR35x5LdFFpSzyw954WzTD+DFyHe3yKlJnG5As=",
        version = "v1.6.3",
    )

    go_repository(
        name = "com_github_wealdtech_go_eth2_wallet_encryptor_keystorev4",
        importpath = "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4",
        sum = "h1:SxrDVSr+oXuT1x8kZt4uWqNCvv5xXEGV9zd7cuSrZS8=",
        version = "v1.1.3",
    )

    go_repository(
        name = "com_github_wealdtech_go_eth2_wallet_types_v2",
        importpath = "github.com/wealdtech/go-eth2-wallet-types/v2",
        sum = "h1:264/meVYWt1wFw6Mtn+xwkZkXjID42gNra4rycoiDXI=",
        version = "v2.8.2",
    )
    go_repository(
        name = "com_github_wercker_journalhook",
        importpath = "github.com/wercker/journalhook",
        sum = "h1:shC1HB1UogxN5Ech3Yqaaxj1X/P656PPCB4RbojIJqc=",
        version = "v0.0.0-20180428041537-5d0a5ae867b3",
    )

    go_repository(
        name = "com_github_whyrusleeping_mdns",
        importpath = "github.com/whyrusleeping/mdns",
        sum = "h1:Y1/FEOpaCpD21WxrmfeIYCFPuVPRCY2XZTWzTNHGw30=",
        version = "v0.0.0-20190826153040-b9b60ed33aa9",
    )

    go_repository(
        name = "com_github_whyrusleeping_multiaddr_filter",
        importpath = "github.com/whyrusleeping/multiaddr-filter",
        sum = "h1:E9S12nwJwEOXe2d6gT6qxdvqMnNq+VnSsKPgm2ZZNds=",
        version = "v0.0.0-20160516205228-e903e4adabd7",
    )
    go_repository(
        name = "com_github_whyrusleeping_timecache",
        importpath = "github.com/whyrusleeping/timecache",
        sum = "h1:lYbXeSvJi5zk5GLKVuid9TVjS9a0OmLIDKTfoZBL6Ow=",
        version = "v0.0.0-20160911033111-cfcb2f1abfee",
    )

    go_repository(
        name = "com_github_willf_bitset",
        importpath = "github.com/willf/bitset",
        sum = "h1:ekJIKh6+YbUIVt9DfNbkR5d6aFcFTLDRyJNAACURBg8=",
        version = "v1.1.3",
    )
    go_repository(
        name = "com_github_wsddn_go_ecdh",
        importpath = "github.com/wsddn/go-ecdh",
        sum = "h1:1cngl9mPEoITZG8s8cVcUy5CeIBYhEESkOB7m6Gmkrk=",
        version = "v0.0.0-20161211032359-48726bab9208",
    )

    go_repository(
        name = "com_github_x_cray_logrus_prefixed_formatter",
        importpath = "github.com/x-cray/logrus-prefixed-formatter",
        sum = "h1:00txxvfBM9muc0jiLIEAkAcIMJzfthRT6usrui8uGmg=",
        version = "v0.5.2",
    )
    go_repository(
        name = "com_github_xdg_scram",
        importpath = "github.com/xdg/scram",
        sum = "h1:u40Z8hqBAAQyv+vATcGgV0YCnDjqSL7/q/JyPhhJSPk=",
        version = "v0.0.0-20180814205039-7eeb5667e42c",
    )
    go_repository(
        name = "com_github_xdg_stringprep",
        importpath = "github.com/xdg/stringprep",
        sum = "h1:d9X0esnoa3dFsV0FG35rAT0RIhYFlPq7MiP+DW89La0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_xiang90_probing",
        importpath = "github.com/xiang90/probing",
        sum = "h1:eY9dn8+vbi4tKz5Qo6v2eYzo7kUS51QINcR5jNpbZS8=",
        version = "v0.0.0-20190116061207-43a291ad63a2",
    )

    go_repository(
        name = "com_github_xlab_treeprint",
        importpath = "github.com/xlab/treeprint",
        sum = "h1:YdYsPAZ2pC6Tow/nPZOPQ96O3hm/ToAkGsPLzedXERk=",
        version = "v0.0.0-20180616005107-d6fb6747feb6",
    )
    go_repository(
        name = "com_github_xordataexchange_crypt",
        importpath = "github.com/xordataexchange/crypt",
        sum = "h1:ESFSdwYZvkeru3RtdrYueztKhOBCSAAzS4Gf+k0tEow=",
        version = "v0.0.3-0.20170626215501-b2862e3d0a77",
    )
    go_repository(
        name = "com_github_xrash_smetrics",
        importpath = "github.com/xrash/smetrics",
        sum = "h1:bAn7/zixMGCfxrRTfdpNzjtPYqr8smhKouy9mxVdGPU=",
        version = "v0.0.0-20201216005158-039620a65673",
    )

    go_repository(
        name = "com_github_xtaci_kcp_go",
        importpath = "github.com/xtaci/kcp-go",
        sum = "h1:TN1uey3Raw0sTz0Fg8GkfM0uH3YwzhnZWQ1bABv5xAg=",
        version = "v5.4.20+incompatible",
    )
    go_repository(
        name = "com_github_xtaci_lossyconn",
        importpath = "github.com/xtaci/lossyconn",
        sum = "h1:J0GxkO96kL4WF+AIT3M4mfUVinOCPgf2uUWYFUzN0sM=",
        version = "v0.0.0-20190602105132-8df528c0c9ae",
    )
    go_repository(
        name = "com_github_yuin_goldmark",
        importpath = "github.com/yuin/goldmark",
        sum = "h1:/vn0k+RBvwlxEmP5E7SZMqNxPhfMVFEJiykr15/0XKM=",
        version = "v1.4.1",
    )

    go_repository(
        name = "com_google_cloud_go",
        importpath = "cloud.google.com/go",
        sum = "h1:Dg9iHVQfrhq82rUNu9ZxUDrJLaxFUe/HlCVaLyRruq8=",
        version = "v0.65.0",
    )

    go_repository(
        name = "com_google_cloud_go_bigquery",
        importpath = "cloud.google.com/go/bigquery",
        sum = "h1:PQcPefKFdaIzjQFbiyOgAqyx8q5djaE7x9Sqe712DPA=",
        version = "v1.8.0",
    )
    go_repository(
        name = "com_google_cloud_go_bigtable",
        importpath = "cloud.google.com/go/bigtable",
        sum = "h1:F4cCmA4nuV84V5zYQ3MKY+M1Cw1avHDuf3S/LcZPA9c=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_google_cloud_go_datastore",
        importpath = "cloud.google.com/go/datastore",
        sum = "h1:/May9ojXjRkPBNVrq+oWLqmWCkr4OU5uRY29bu0mRyQ=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_google_cloud_go_firestore",
        importpath = "cloud.google.com/go/firestore",
        sum = "h1:9x7Bx0A9R5/M9jibeJeZWqjeVEIxYW9fZYqB9a70/bY=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_google_cloud_go_pubsub",
        importpath = "cloud.google.com/go/pubsub",
        sum = "h1:ukjixP1wl0LpnZ6LWtZJ0mX5tBmjp1f8Sqer8Z2OMUU=",
        version = "v1.3.1",
    )
    go_repository(
        name = "com_google_cloud_go_storage",
        importpath = "cloud.google.com/go/storage",
        sum = "h1:STgFzyU5/8miMl0//zKh2aQeTyeaUH3WN9bSUiJ09bA=",
        version = "v1.10.0",
    )
    go_repository(
        name = "com_lukechampine_blake3",
        importpath = "lukechampine.com/blake3",
        sum = "h1:GgRMhmdsuK8+ii6UZFDL8Nb+VyMwadAgcJyfYHxG6n0=",
        version = "v1.1.7",
    )

    go_repository(
        name = "com_shuralyov_dmitri_app_changes",
        importpath = "dmitri.shuralyov.com/app/changes",
        sum = "h1:hJiie5Bf3QucGRa4ymsAUOxyhYwGEz1xrsVk0P8erlw=",
        version = "v0.0.0-20180602232624-0a106ad413e3",
    )

    go_repository(
        name = "com_shuralyov_dmitri_gpu_mtl",
        importpath = "dmitri.shuralyov.com/gpu/mtl",
        sum = "h1:VpgP7xuJadIUuKccphEpTJnWhS2jkQyMt6Y7pJCD7fY=",
        version = "v0.0.0-20190408044501-666a987793e9",
    )
    go_repository(
        name = "com_shuralyov_dmitri_html_belt",
        importpath = "dmitri.shuralyov.com/html/belt",
        sum = "h1:SPOUaucgtVls75mg+X7CXigS71EnsfVUK/2CgVrwqgw=",
        version = "v0.0.0-20180602232347-f7d459c86be0",
    )
    go_repository(
        name = "com_shuralyov_dmitri_service_change",
        importpath = "dmitri.shuralyov.com/service/change",
        sum = "h1:GvWw74lx5noHocd+f6HBMXK6DuggBB1dhVkuGZbv7qM=",
        version = "v0.0.0-20181023043359-a85b471d5412",
    )
    go_repository(
        name = "com_shuralyov_dmitri_state",
        importpath = "dmitri.shuralyov.com/state",
        sum = "h1:ivON6cwHK1OH26MZyWDCnbTRZZf0IhNsENoNAKFS1g4=",
        version = "v0.0.0-20180228185332-28bcc343414c",
    )

    go_repository(
        name = "com_sourcegraph_sourcegraph_appdash",
        importpath = "sourcegraph.com/sourcegraph/appdash",
        sum = "h1:ucqkfpjg9WzSUubAO62csmucvxl4/JeW3F4I4909XkM=",
        version = "v0.0.0-20190731080439-ebfcffb1b5c0",
    )
    go_repository(
        name = "com_sourcegraph_sourcegraph_go_diff",
        importpath = "sourcegraph.com/sourcegraph/go-diff",
        sum = "h1:eTiIR0CoWjGzJcnQ3OkhIl/b9GJovq4lSAVRt0ZFEG8=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_sourcegraph_sqs_pbtypes",
        importpath = "sourcegraph.com/sqs/pbtypes",
        sum = "h1:JPJh2pk3+X4lXAkZIk2RuE/7/FoK9maXw+TNPJhVS/c=",
        version = "v0.0.0-20180604144634-d3ebe8f20ae4",
    )

    go_repository(
        name = "in_gopkg_alecthomas_kingpin_v2",
        importpath = "gopkg.in/alecthomas/kingpin.v2",
        sum = "h1:jMFz6MfLP0/4fUyZle81rXUoxOBFi19VUFKVDOQfozc=",
        version = "v2.2.6",
    )
    go_repository(
        name = "in_gopkg_bsm_ratelimit_v1",
        importpath = "gopkg.in/bsm/ratelimit.v1",
        sum = "h1:stTHdEoWg1pQ8riaP5ROrjS6zy6wewH/Q2iwnLCQUXY=",
        version = "v1.0.0-20160220154919-db14e161995a",
    )
    go_repository(
        name = "in_gopkg_cenkalti_backoff_v1",
        importpath = "gopkg.in/cenkalti/backoff.v1",
        sum = "h1:Arh75ttbsvlpVA7WtVpH4u9h6Zl46xuptxqLxPiSo4Y=",
        version = "v1.1.0",
    )

    go_repository(
        name = "in_gopkg_check_v1",
        importpath = "gopkg.in/check.v1",
        sum = "h1:Hei/4ADfdWqJk1ZMxUNpqntNwaWcugrBjAiHlqqRiVk=",
        version = "v1.0.0-20201130134442-10cb98267c6c",
    )
    go_repository(
        name = "in_gopkg_cheggaaa_pb_v1",
        importpath = "gopkg.in/cheggaaa/pb.v1",
        sum = "h1:Ev7yu1/f6+d+b3pi5vPdRPc6nNtP1umSfcWiEfRqv6I=",
        version = "v1.0.25",
    )
    go_repository(
        name = "in_gopkg_d4l3k_messagediff_v1",
        importpath = "gopkg.in/d4l3k/messagediff.v1",
        sum = "h1:70AthpjunwzUiarMHyED52mj9UwtAnE89l1Gmrt3EU0=",
        version = "v1.2.1",
    )
    go_repository(
        name = "in_gopkg_errgo_v2",
        importpath = "gopkg.in/errgo.v2",
        sum = "h1:0vLT13EuvQ0hNvakwLuFZ/jYrLp5F3kcWHXdRggjCE8=",
        version = "v2.1.0",
    )
    go_repository(
        name = "in_gopkg_fsnotify_v1",
        importpath = "gopkg.in/fsnotify.v1",
        sum = "h1:xOHLXZwVvI9hhs+cLKq5+I5onOuwQLhQwiu63xxlHs4=",
        version = "v1.4.7",
    )
    go_repository(
        name = "in_gopkg_gcfg_v1",
        importpath = "gopkg.in/gcfg.v1",
        sum = "h1:m8OOJ4ccYHnx2f4gQwpno8nAX5OGOh7RLaaz0pj3Ogs=",
        version = "v1.2.3",
    )

    go_repository(
        name = "in_gopkg_inf_v0",
        importpath = "gopkg.in/inf.v0",
        sum = "h1:73M5CoZyi3ZLMOyDlQh031Cx6N9NDJ2Vvfl76EDAgDc=",
        version = "v0.9.1",
    )

    go_repository(
        name = "in_gopkg_ini_v1",
        importpath = "gopkg.in/ini.v1",
        sum = "h1:AQvPpx3LzTDM0AjnIRlVFwFFGC+npRopjZxLJj6gdno=",
        version = "v1.51.0",
    )
    go_repository(
        name = "in_gopkg_jcmturner_aescts_v1",
        importpath = "gopkg.in/jcmturner/aescts.v1",
        sum = "h1:cVVZBK2b1zY26haWB4vbBiZrfFQnfbTVrE3xZq6hrEw=",
        version = "v1.0.1",
    )
    go_repository(
        name = "in_gopkg_jcmturner_dnsutils_v1",
        importpath = "gopkg.in/jcmturner/dnsutils.v1",
        sum = "h1:cIuC1OLRGZrld+16ZJvvZxVJeKPsvd5eUIvxfoN5hSM=",
        version = "v1.0.1",
    )
    go_repository(
        name = "in_gopkg_jcmturner_goidentity_v3",
        importpath = "gopkg.in/jcmturner/goidentity.v3",
        sum = "h1:1duIyWiTaYvVx3YX2CYtpJbUFd7/UuPYCfgXtQ3VTbI=",
        version = "v3.0.0",
    )
    go_repository(
        name = "in_gopkg_jcmturner_gokrb5_v7",
        importpath = "gopkg.in/jcmturner/gokrb5.v7",
        sum = "h1:a9tsXlIDD9SKxotJMK3niV7rPZAJeX2aD/0yg3qlIrg=",
        version = "v7.5.0",
    )
    go_repository(
        name = "in_gopkg_jcmturner_rpc_v1",
        importpath = "gopkg.in/jcmturner/rpc.v1",
        sum = "h1:QHIUxTX1ISuAv9dD2wJ9HWQVuWDX/Zc0PfeC2tjc4rU=",
        version = "v1.1.0",
    )

    go_repository(
        name = "in_gopkg_natefinch_npipe_v2",
        importpath = "gopkg.in/natefinch/npipe.v2",
        sum = "h1:+JknDZhAj8YMt7GC73Ei8pv4MzjDUNPHgQWJdtMAaDU=",
        version = "v2.0.0-20160621034901-c1b8fa8bdcce",
    )
    go_repository(
        name = "in_gopkg_olebedev_go_duktape_v3",
        importpath = "gopkg.in/olebedev/go-duktape.v3",
        sum = "h1:a6cXbcDDUkSBlpnkWV1bJ+vv3mOgQEltEJ2rPxroVu0=",
        version = "v3.0.0-20200619000410-60c24ae608a6",
    )
    go_repository(
        name = "in_gopkg_redis_v4",
        importpath = "gopkg.in/redis.v4",
        sum = "h1:y3XbwQAiHwgNLUng56mgWYK39vsPqo8sT84XTEcxjr0=",
        version = "v4.2.4",
    )
    go_repository(
        name = "in_gopkg_resty_v1",
        importpath = "gopkg.in/resty.v1",
        sum = "h1:CuXP0Pjfw9rOuY6EP+UvtNvt5DSqHpIxILZKT/quCZI=",
        version = "v1.12.0",
    )

    go_repository(
        name = "in_gopkg_src_d_go_cli_v0",
        importpath = "gopkg.in/src-d/go-cli.v0",
        sum = "h1:mXa4inJUuWOoA4uEROxtJ3VMELMlVkIxIfcR0HBekAM=",
        version = "v0.0.0-20181105080154-d492247bbc0d",
    )
    go_repository(
        name = "in_gopkg_src_d_go_log_v1",
        importpath = "gopkg.in/src-d/go-log.v1",
        sum = "h1:heWvX7J6qbGWbeFS/aRmiy1eYaT+QMV6wNvHDyMjQV4=",
        version = "v1.0.1",
    )
    go_repository(
        name = "in_gopkg_tomb_v1",
        importpath = "gopkg.in/tomb.v1",
        sum = "h1:uRGJdciOHaEIrze2W8Q3AKkepLTh2hOroT7a+7czfdQ=",
        version = "v1.0.0-20141024135613-dd632973f1e7",
    )

    go_repository(
        name = "in_gopkg_urfave_cli_v1",
        importpath = "gopkg.in/urfave/cli.v1",
        sum = "h1:NdAVW6RYxDif9DhDHaAortIu956m2c0v+09AZBPTbE0=",
        version = "v1.20.0",
    )
    go_repository(
        name = "in_gopkg_warnings_v0",
        importpath = "gopkg.in/warnings.v0",
        sum = "h1:wFXVbFY8DY5/xOe1ECiWdKCzZlxgshcYVNkBHstARME=",
        version = "v0.1.2",
    )

    go_repository(
        name = "in_gopkg_yaml_v2",
        importpath = "gopkg.in/yaml.v2",
        sum = "h1:D8xgwECY7CYvx+Y2n4sBz93Jn9JRvxdiyyo8CTfuKaY=",
        version = "v2.4.0",
    )

    go_repository(
        name = "in_gopkg_yaml_v3",
        importpath = "gopkg.in/yaml.v3",
        sum = "h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=",
        version = "v3.0.1",
    )
    go_repository(
        name = "io_etcd_go_bbolt",
        importpath = "go.etcd.io/bbolt",
        sum = "h1:XAzx9gjCb0Rxj7EoqcClPD1d5ZBxZJk0jbuoPHenBt0=",
        version = "v1.3.5",
    )
    go_repository(
        name = "io_etcd_go_etcd",
        importpath = "go.etcd.io/etcd",
        sum = "h1:VcrIfasaLFkyjk6KNlXQSzO+B0fZcnECiDrKJsfxka0=",
        version = "v0.0.0-20191023171146-3cf2f69b5738",
    )

    go_repository(
        name = "io_k8s_api",
        build_file_proto_mode = "disable_global",
        importpath = "k8s.io/api",
        sum = "h1:2AJaUQdgUZLoDZHrun21PW2Nx9+ll6cUzvn3IKhSIn0=",
        version = "v0.18.3",
    )
    go_repository(
        name = "io_k8s_apimachinery",
        build_file_proto_mode = "disable_global",
        importpath = "k8s.io/apimachinery",
        sum = "h1:pOGcbVAhxADgUYnjS08EFXs9QMl8qaH5U4fr5LGUrSk=",
        version = "v0.18.3",
    )
    go_repository(
        name = "io_k8s_client_go",
        build_extra_args = ["-exclude=vendor"],
        build_naming_convention = "go_default_library",
        importpath = "k8s.io/client-go",
        sum = "h1:QaJzz92tsN67oorwzmoB0a9r9ZVHuD5ryjbCKP0U22k=",
        version = "v0.18.3",
    )
    go_repository(
        name = "io_k8s_gengo",
        importpath = "k8s.io/gengo",
        sum = "h1:4s3/R4+OYYYUKptXPhZKjQ04WJ6EhQQVFdjOFvCazDk=",
        version = "v0.0.0-20190128074634-0689ccc1d7d6",
    )

    go_repository(
        name = "io_k8s_klog",
        importpath = "k8s.io/klog",
        sum = "h1:Pt+yjF5aB1xDSVbau4VsWe+dQNzA0qv1LlXdC2dF6Q8=",
        version = "v1.0.0",
    )
    go_repository(
        name = "io_k8s_klog_v2",
        importpath = "k8s.io/klog/v2",
        sum = "h1:WmkrnW7fdrm0/DMClc+HIxtftvxVIPAhlVwMQo5yLco=",
        version = "v2.3.0",
    )
    go_repository(
        name = "io_k8s_kube_openapi",
        importpath = "k8s.io/kube-openapi",
        sum = "h1:Oh3Mzx5pJ+yIumsAD0MOECPVeXsVot0UkiaCGVyfGQY=",
        version = "v0.0.0-20200410145947-61e04a5be9a6",
    )

    go_repository(
        name = "io_k8s_sigs_structured_merge_diff_v3",
        importpath = "sigs.k8s.io/structured-merge-diff/v3",
        sum = "h1:dOmIZBMfhcHS09XZkMyUgkq5trg3/jRyJYFZUiaOp8E=",
        version = "v3.0.0",
    )

    go_repository(
        name = "io_k8s_sigs_yaml",
        importpath = "sigs.k8s.io/yaml",
        sum = "h1:kr/MCeFWJWTwyaHoR9c8EjH9OumOmoF9YGiZd7lFm/Q=",
        version = "v1.2.0",
    )
    go_repository(
        name = "io_k8s_utils",
        importpath = "k8s.io/utils",
        sum = "h1:ZtTUW5+ZWaoqjR3zOpRa7oFJ5d4aA22l4me/xArfOIc=",
        version = "v0.0.0-20200520001619-278ece378a50",
    )
    go_repository(
        name = "io_opencensus_go",
        importpath = "go.opencensus.io",
        sum = "h1:gqCw0LfLxScz8irSi8exQc7fyQ0fKQU/qnC/X8+V/1M=",
        version = "v0.23.0",
    )
    go_repository(
        name = "io_opencensus_go_contrib_exporter_jaeger",
        importpath = "contrib.go.opencensus.io/exporter/jaeger",
        sum = "h1:yGBYzYMewVL0yO9qqJv3Z5+IRhPdU7e9o/2oKpX4YvI=",
        version = "v0.2.1",
    )
    go_repository(
        name = "io_opentelemetry_go_proto_otlp",
        importpath = "go.opentelemetry.io/proto/otlp",
        sum = "h1:rwOQPCuKAKmwGKq2aVNnYIibI6wnV7EvzgfTCzcdGg8=",
        version = "v0.7.0",
    )

    go_repository(
        name = "io_rsc_binaryregexp",
        importpath = "rsc.io/binaryregexp",
        sum = "h1:HfqmD5MEmC0zvwBuF187nq9mdnXjXsSivRiXN7SmRkE=",
        version = "v0.2.0",
    )
    go_repository(
        name = "io_rsc_pdf",
        importpath = "rsc.io/pdf",
        sum = "h1:k1MczvYDUvJBe93bYd7wrZLLUEcLZAuF824/I4e5Xr4=",
        version = "v0.1.1",
    )
    go_repository(
        name = "io_rsc_quote_v3",
        importpath = "rsc.io/quote/v3",
        sum = "h1:9JKUTTIUgS6kzR9mK1YuGKv6Nl+DijDNIc0ghT58FaY=",
        version = "v3.1.0",
    )
    go_repository(
        name = "io_rsc_sampler",
        importpath = "rsc.io/sampler",
        sum = "h1:7uVkIFmeBqHfdjD+gZwtXXI+RODJ2Wc4O7MPEh/QiW4=",
        version = "v1.3.0",
    )
    go_repository(
        name = "org_apache_git_thrift_git",
        importpath = "git.apache.org/thrift.git",
        sum = "h1:OR8VhtwhcAI3U48/rzBsVOuHi0zDPzYI1xASVcdSgR8=",
        version = "v0.0.0-20180902110319-2566ecd5d999",
    )

    go_repository(
        name = "org_collectd",
        importpath = "collectd.org",
        sum = "h1:iNBHGw1VvPJxH2B6RiFWFZ+vsjo1lCdRszBeOuwGi00=",
        version = "v0.3.0",
    )
    go_repository(
        name = "org_go4",
        importpath = "go4.org",
        sum = "h1:+hE86LblG4AyDgwMCLTE6FOlM9+qjHSYS+rKqxUVdsM=",
        version = "v0.0.0-20180809161055-417644f6feb5",
    )
    go_repository(
        name = "org_go4_grpc",
        importpath = "grpc.go4.org",
        sum = "h1:tmXTu+dfa+d9Evp8NpJdgOy6+rt8/x4yG7qPBrtNfLY=",
        version = "v0.0.0-20170609214715-11d0a25b4919",
    )

    go_repository(
        name = "org_golang_google_api",
        importpath = "google.golang.org/api",
        sum = "h1:k40adF3uR+6x/+hO5Dh4ZFUqFp67vxvbpafFiJxl10A=",
        version = "v0.34.0",
    )
    go_repository(
        name = "org_golang_google_appengine",
        importpath = "google.golang.org/appengine",
        sum = "h1:FZR1q0exgwxzPzp/aF+VccGrSfxfPpkBqjIIEq3ru6c=",
        version = "v1.6.7",
    )
    go_repository(
        name = "org_golang_google_genproto",
        importpath = "google.golang.org/genproto",
        sum = "h1:KMgpo2lWy1vfrYjtxPAzR0aNWeAR1UdQykt6sj/hpBY=",
        version = "v0.0.0-20210426193834-eac7f76ac494",
    )

    go_repository(
        name = "org_golang_google_grpc",
        build_file_proto_mode = "disable",
        importpath = "google.golang.org/grpc",
        sum = "h1:AGJ0Ih4mHjSeibYkFGh1dD9KJ/eOtZ93I6hoHhukQ5Q=",
        version = "v1.40.0",
    )
    go_repository(
        name = "org_golang_google_grpc_cmd_protoc_gen_go_grpc",
        importpath = "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
        sum = "h1:lQ+dE99pFsb8osbJB3oRfE5eW4Hx6a/lZQr8Jh+eoT4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "org_golang_google_protobuf",
        importpath = "google.golang.org/protobuf",
        sum = "h1:w43yiav+6bVFTBQFZX0r7ipe9JQ1QsbMgHwbBziscLw=",
        version = "v1.28.0",
    )
    go_repository(
        name = "org_golang_x_build",
        importpath = "golang.org/x/build",
        sum = "h1:E2M5QgjZ/Jg+ObCQAudsXxuTsLj7Nl5RV/lZcQZmKSo=",
        version = "v0.0.0-20190111050920-041ab4dc3f9d",
    )

    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:sK3txAijHtOK88l68nt020reeT1ZdKLIYetKl95FzVY=",
        version = "v0.0.0-20220622213112-05595931fe9d",
    )
    go_repository(
        name = "org_golang_x_exp",
        importpath = "golang.org/x/exp",
        sum = "h1:rxKZ2gOnYxjfmakvUUqh9Gyb6KXfrj7JWTxORTYqb0E=",
        version = "v0.0.0-20220426173459-3bcf042a4bf5",
    )
    go_repository(
        name = "org_golang_x_image",
        importpath = "golang.org/x/image",
        sum = "h1:+qEpEAPhDZ1o0x3tHzZTQDArnOixOzGD9HUJfcg0mb4=",
        version = "v0.0.0-20190802002840-cff245a6509b",
    )

    go_repository(
        name = "org_golang_x_lint",
        importpath = "golang.org/x/lint",
        sum = "h1:2M3HP5CCK1Si9FQhwnzYhXdG6DXeebvUHFpre8QvbyI=",
        version = "v0.0.0-20201208152925-83fdc39ff7b5",
    )
    go_repository(
        name = "org_golang_x_mobile",
        importpath = "golang.org/x/mobile",
        sum = "h1:OVJ6QQUBAesB8CZijKDSsXX7xYVtUhrkY0gwMfbi4p4=",
        version = "v0.0.0-20200801112145-973feb4309de",
    )

    go_repository(
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:6zppjxzCulZykYSLyVDYbneBfbaBIQPYMevg0bEwv2s=",
        version = "v0.6.0-dev.0.20220419223038-86c51ed26bb4",
    )

    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:TsQ7F31D3bUCLeqPT0u+yjp1guoArKaNKmCr22PYgTQ=",
        version = "v0.0.0-20220624214902-1bab6f366d9e",
    )
    go_repository(
        name = "org_golang_x_oauth2",
        importpath = "golang.org/x/oauth2",
        sum = "h1:clP8eMhB30EHdc0bd2Twtq6kgU7yl5ub2cQLSdrv1Dg=",
        version = "v0.0.0-20220223155221-ee480838109b",
    )
    go_repository(
        name = "org_golang_x_perf",
        importpath = "golang.org/x/perf",
        sum = "h1:xYq6+9AtI+xP3M4r0N1hCkHrInHDBohhquRgx9Kk6gI=",
        version = "v0.0.0-20180704124530-6e6d33e29852",
    )

    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:Ax0t5p6N38Ga0dThY21weqDEyz2oklo4IvDkpigvkD8=",
        version = "v0.0.0-20220601150217-0de741cfad7f",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:xdsejrW/0Wf2diT5CPp3XmKUNbr7Xvw8kYilQ+6qjRY=",
        version = "v0.0.0-20220702020025-31831981b65f",
    )
    go_repository(
        name = "org_golang_x_term",
        importpath = "golang.org/x/term",
        sum = "h1:JGgROgKl9N8DuW20oFS5gxc+lE67/N3FcwmBPMe7ArY=",
        version = "v0.0.0-20210927222741-03fcf44c2211",
    )

    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:olpwvP2KacW1ZWvsR7uQhoyTYvKAupfQrRGBFM352Gk=",
        version = "v0.3.7",
    )
    go_repository(
        name = "org_golang_x_time",
        importpath = "golang.org/x/time",
        sum = "h1:O8mE0/t419eoIwhTFpKVkHiTs/Igowgfkj25AcZrtiE=",
        version = "v0.0.0-20210220033141-f8bda1e9f3ba",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:loJ25fNOEhSXfHrpoGj91eCUThwdNX6u24rO1xnNteY=",
        version = "v0.1.11",
    )

    go_repository(
        name = "org_golang_x_xerrors",
        importpath = "golang.org/x/xerrors",
        sum = "h1:5Pf6pFKu98ODmgnpvkJ3kFUOQGGLIzLIkbzUHp47618=",
        version = "v0.0.0-20220517211312-f3a8303e98df",
    )

    go_repository(
        name = "org_gonum_v1_gonum",
        importpath = "gonum.org/v1/gonum",
        sum = "h1:DJy6UzXbahnGUf1ujUNkh/NEtK14qMo2nvlBPs4U5yw=",
        version = "v0.6.0",
    )
    go_repository(
        name = "org_gonum_v1_netlib",
        importpath = "gonum.org/v1/netlib",
        sum = "h1:OE9mWmgKkjJyEmDAAtGMPjXu+YNeGvK9VTSHY6+Qihc=",
        version = "v0.0.0-20190313105609-8cb42192e0e0",
    )
    go_repository(
        name = "org_gonum_v1_plot",
        importpath = "gonum.org/v1/plot",
        sum = "h1:Qh4dB5D/WpoUUp3lSod7qgoyEHbDGPUWjIbnqdqqe1k=",
        version = "v0.0.0-20190515093506-e2840ee46a6b",
    )
    go_repository(
        name = "org_uber_go_atomic",
        importpath = "go.uber.org/atomic",
        sum = "h1:ECmE8Bn/WFTYwEW/bpKD3M8VtR/zQVbavAoalC1PYyE=",
        version = "v1.9.0",
    )

    go_repository(
        name = "org_uber_go_automaxprocs",
        build_directives = [
            # Do not use this library directly.
            # Rather, load maxprocs from github.com/prysmaticlabs/runtime/maxprocs.
            "gazelle:go_visibility @prysm//runtime/maxprocs:__pkg__",
        ],
        importpath = "go.uber.org/automaxprocs",
        sum = "h1:II28aZoGdaglS5vVNnspf28lnZpXScxtIozx1lAjdb0=",
        version = "v1.3.0",
    )
    go_repository(
        name = "org_uber_go_goleak",
        importpath = "go.uber.org/goleak",
        sum = "h1:gZAh5/EyT/HQwlpkCy6wTpqfH9H8Lz8zbm3dZh+OyzA=",
        version = "v1.1.12",
    )
    go_repository(
        name = "org_uber_go_multierr",
        importpath = "go.uber.org/multierr",
        sum = "h1:dg6GjLku4EH+249NNmoIciG9N/jURbDG+pFlTkhzIC8=",
        version = "v1.8.0",
    )
    go_repository(
        name = "org_uber_go_tools",
        importpath = "go.uber.org/tools",
        sum = "h1:0mgffUl7nfd+FpvXMVz4IDEaUSmT1ysygQC7qYo7sG4=",
        version = "v0.0.0-20190618225709-2cfd321de3ee",
    )
    go_repository(
        name = "org_uber_go_zap",
        importpath = "go.uber.org/zap",
        sum = "h1:WefMeulhovoZ2sYXz7st6K0sLj7bBhpiFaud4r4zST8=",
        version = "v1.21.0",
    )
    go_repository(
        name = "tools_gotest",
        importpath = "gotest.tools",
        sum = "h1:VsBPFP1AI068pPrMxtb/S8Zkgf9xEmTLJjfM+P5UIEo=",
        version = "v2.2.0+incompatible",
    )

    # Note: go_repository is already wrapped with maybe!
    maybe(
        git_repository,
        name = "vaticle_bazel_distribution",
        commit = "96424c85195a97dad81f69fdbbef2e1574bf8801",
        remote = "https://github.com/vaticle/bazel-distribution",
        shallow_since = "1569509514 +0300",
    )
