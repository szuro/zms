$version='0.5.1'
$commit=$(git log -n 1 --pretty=format:"%H")
$time=(Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ss.fffK")


$env:GOOS="linux"
$env:GOARCH="amd64"

go build -trimpath -ldflags="-X szuro.net/zms/internal/config.Version=$version -X szuro.net/zms/internal/config.Commit=$commit -X szuro.net/zms/internal/config.BuildDate=$time" -o zmsd ./cmd/zmsd
