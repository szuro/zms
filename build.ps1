$version='0.5.0'
$commit=$(git log -n 1 --pretty=format:"%H")
$time=(Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ss.fffK")


$env:GOOS="linux"
$env:GOARCH="amd64"

go build -trimpath -ldflags="-X zms.Version=$version -X zms.Commit=$commit -X zms.BuildDate=$time" -o zmsd .
