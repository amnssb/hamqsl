# QSL login / 2FA smoke test v2 (ASCII only: PS 5.1 reads .ps1 as ANSI and would eat the next line)
# Two isolated runs: functional chain first, then rate limit (buckets must be fresh)
$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..\backend')).Path
$base = 'http://127.0.0.1:18080/api'
$results = New-Object System.Collections.Generic.List[string]
$proc = $null

function Ok($name, $cond, $detail) {
  $mark = if ($cond) { 'PASS' } else { 'FAIL' }
  $results.Add(("{0}  {1}  {2}" -f $mark, $name, $detail))
}

# ---------- TOTP ----------
function Base32Decode([string]$s) {
  $alpha = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
  $s = $s.ToUpper().TrimEnd('=')
  $bits = ''
  foreach ($c in $s.ToCharArray()) {
    $idx = $alpha.IndexOf($c)
    if ($idx -lt 0) { throw "bad base32 char $c" }
    $bits += [Convert]::ToString($idx, 2).PadLeft(5, '0')
  }
  $out = New-Object System.Collections.Generic.List[byte]
  for ($i = 0; $i + 8 -le $bits.Length; $i += 8) {
    $out.Add([Convert]::ToByte($bits.Substring($i, 8), 2))
  }
  return , $out.ToArray()
}

function TotpAtStep([string]$secret, [int64]$step) {
  [byte[]]$key = Base32Decode $secret
  $msg = [BitConverter]::GetBytes($step)
  [Array]::Reverse($msg)
  $h = [System.Security.Cryptography.HMACSHA1]::Create()
  $h.Key = $key
  $hash = $h.ComputeHash($msg)
  $off = [int]$hash[$hash.Length - 1] -band 0x0F
  # use multiplication: -shl/-bor on byte values truncates in PowerShell
  $b0 = [int]$hash[$off] -band 0x7F
  $b1 = [int]$hash[$off + 1]
  $b2 = [int]$hash[$off + 2]
  $b3 = [int]$hash[$off + 3]
  $bin = ($b0 * 16777216) + ($b1 * 65536) + ($b2 * 256) + $b3
  return ('{0:D6}' -f ($bin % 1000000))
}

function TotpNow([string]$secret) {
  return (TotpAtStep $secret ([int64]([DateTimeOffset]::UtcNow.ToUnixTimeSeconds() / 30)))
}

# ---------- HTTP (curl; PS 5.1 mangles quotes passed to native exe) ----------
$respFile = Join-Path $root '.tmp\resp.json'
$bodyFile = Join-Path $root '.tmp\req.json'
function Call($method, $path, $body, $token, $xff) {
  $a = @('-s', '-o', $respFile, '-w', '%{http_code}', '--max-time', '30', '-X', $method)
  if ($token) { $a += @('-H', "Authorization: Bearer $token") }
  if ($xff) { $a += @('-H', "X-Forwarded-For: $xff") }
  if ($null -ne $body) {
    # write body to a file first: PS 5.1 eats quotes when passing JSON to curl
    $json = $body | ConvertTo-Json -Compress
    [System.IO.File]::WriteAllText($bodyFile, $json, (New-Object System.Text.UTF8Encoding($false)))
    $a += @('-H', 'Content-Type: application/json', '--data-binary', "@$bodyFile")
  }
  $a += "$base$path"
  $code = & curl.exe @a
  $n = 0
  [void][int]::TryParse((($code | Out-String).Trim()), [ref]$n)
  $obj = $null
  if (Test-Path $respFile) {
    try { $obj = (Get-Content $respFile -Raw -Encoding UTF8) | ConvertFrom-Json } catch {}
  }
  return @{ code = $n; obj = $obj }
}

function StartServer([string]$dbFile, [string]$outName) {
  $env:DB_PATH = "./.tmp/$dbFile"
  $env:SERVER_PORT = '18080'
  $env:GIN_MODE = 'release'
  $env:ADMIN_USER = 'admin'
  $env:ADMIN_PASS = 'SmokePass-123456'
  $env:WEBAUTHN_RP_ID = 'localhost'
  $env:WEBAUTHN_ORIGINS = 'http://localhost:18080'
  $env:ADMIN_RESET = ''
  $script:proc = Start-Process -FilePath "$root\.tmp\qsl-test.exe" -WorkingDirectory $root -PassThru `
    -RedirectStandardOutput "$root\.tmp\$outName.out" -RedirectStandardError "$root\.tmp\$outName.err"
  for ($i = 0; $i -lt 40; $i++) {
    Start-Sleep -Milliseconds 250
    $r = Call 'GET' '/auth/capabilities' $null $null $null
    if ($r.code -eq 200) { return }
  }
  Get-Content "$root\.tmp\$outName.err" -ErrorAction SilentlyContinue | Select-Object -First 20 | Write-Output
  throw 'server did not start'
}

function StopServer {
  if ($script:proc -and -not $script:proc.HasExited) { Stop-Process -Id $script:proc.Id -Force }
  $script:proc = $null
  Start-Sleep -Milliseconds 400
}

function WaitNextStep {
  $now = [int64]([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())
  Start-Sleep -Seconds (30 - ($now % 30) + 1)
}

# ---------- build ----------
Get-ChildItem "$root\.tmp" -Filter 'qsl-test*' -ErrorAction SilentlyContinue | Remove-Item -Force -Recurse -ErrorAction SilentlyContinue
Get-ChildItem "$root\.tmp" -Filter 'smoke*.db' -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path "$root\.tmp" | Out-Null
$env:CGO_ENABLED = '1'
Push-Location $root
go build -o .tmp\qsl-test.exe ./cmd/server
if ($LASTEXITCODE -ne 0) { throw 'build failed' }

try {
  # ===== self test: RFC 6238 vector, T=59 (step 1) -> 287082 =====
  $c = TotpAtStep 'GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ' 1
  Ok 'totp self-test (RFC6238)' ($c -eq '287082') ("got=" + $c + " n=" + @($c).Count + " type=" + $c.GetType().Name)

  # ===== run 1: functional =====
  StartServer 'smoke1.db' 'srv1'

  $cap = Call 'GET' '/auth/capabilities' $null $null $null
  Ok 'capabilities' ($cap.code -eq 200 -and $cap.obj.data.passkey -eq $true) ("code=" + $cap.code + " passkey=" + $cap.obj.data.passkey)

  $login = Call 'POST' '/auth/login' @{ username = 'admin'; password = 'SmokePass-123456' } $null $null
  $tok = $login.obj.data.access_token
  Ok 'login ok' ($login.code -eq 200 -and $tok) ("code=" + $login.code)

  $me = Call 'GET' '/auth/me' $null $tok $null
  Ok 'me' ($me.code -eq 200 -and $me.obj.data.username -eq 'admin') ("code=" + $me.code + " user=" + $me.obj.data.username)

  $bad = Call 'GET' '/auth/me' $null 'not-a-token' $null
  Ok 'bad token rejected' ($bad.code -eq 401) ("code=" + $bad.code)

  $exp = Call 'GET' '/admin/export' $null $tok $null
  Ok 'export allowed before 2fa' ($exp.code -eq 200) ("code=" + $exp.code)

  $setup = Call 'POST' '/auth/totp/setup' @{} $tok $null
  $secret = $setup.obj.data.secret
  Ok 'totp setup' ($setup.code -eq 200 -and $secret) ("code=" + $setup.code)

  $code1 = TotpNow $secret
  $enable = Call 'POST' '/auth/totp/enable' @{ code = $code1 } $tok $null
  $codes = $enable.obj.data.recovery_codes
  Ok 'totp enable' ($enable.code -eq 200 -and $codes.Count -eq 10) ("code=" + $enable.code + " recovery=" + $codes.Count)

  $login2 = Call 'POST' '/auth/login' @{ username = 'admin'; password = 'SmokePass-123456' } $null $null
  $ticket = $login2.obj.data.ticket
  Ok 'login returns mfa_required' ($login2.code -eq 200 -and $login2.obj.data.mfa_required -eq $true -and $ticket) `
    ("code=" + $login2.code + " methods=" + ($login2.obj.data.methods -join ','))

  $exp2 = Call 'GET' '/admin/export' $null $tok $null
  Ok 'export blocked after 2fa' ($exp2.code -eq 403) ("code=" + $exp2.code)

  $replay = Call 'POST' '/auth/mfa/totp' @{ ticket = $ticket; code = $code1 } $null $null
  Ok 'totp replay rejected' ($replay.code -eq 401) ("code=" + $replay.code)

  WaitNextStep
  $code2 = TotpNow $secret
  $mfa = Call 'POST' '/auth/mfa/totp' @{ ticket = $ticket; code = $code2 } $null $null
  $tok2 = $mfa.obj.data.access_token
  Ok 'totp verify -> tokens' ($mfa.code -eq 200 -and $tok2) ("code=" + $mfa.code)

  $exp3 = Call 'GET' '/admin/export' $null $tok2 $null
  Ok 'export allowed with mfa token' ($exp3.code -eq 200) ("code=" + $exp3.code)

  $rename = Call 'POST' '/auth/username' @{ username = 'ba4slt'; password = 'SmokePass-123456' } $tok2 $null
  $tok3 = $rename.obj.data.access_token
  Ok 'rename ok' ($rename.code -eq 200 -and $tok3) ("code=" + $rename.code)

  $oldAfter = Call 'GET' '/auth/me' $null $tok2 $null
  Ok 'old token revoked by rename' ($oldAfter.code -eq 401) ("code=" + $oldAfter.code)

  $newAfter = Call 'GET' '/auth/me' $null $tok3 $null
  Ok 'new token works' ($newAfter.code -eq 200 -and $newAfter.obj.data.username -eq 'ba4slt') ("code=" + $newAfter.code + " user=" + $newAfter.obj.data.username)

  StopServer

  # ===== run 2: rate limit (fresh process, clean buckets) =====
  StartServer 'smoke2.db' 'srv2'

  $seq = @()
  for ($i = 1; $i -le 6; $i++) {
    $r = Call 'POST' '/auth/login' @{ username = 'admin'; password = "wrong-$i" } $null "203.0.113.$i"
    $seq += $r.code
  }
  Ok 'rotating XFF still hits account bucket' (($seq -join ',') -eq '401,401,401,401,401,429') ("got=" + ($seq -join ','))

  $locked = Call 'POST' '/auth/login' @{ username = 'admin'; password = 'SmokePass-123456' } $null '203.0.113.200'
  Ok 'locked: correct password also refused' ($locked.code -eq 429 -or $locked.code -eq 423) ("code=" + $locked.code)

  # spoofing a loopback XFF must not bypass (RealClientIP ignores XFF)
  $loop = Call 'POST' '/auth/login' @{ username = 'admin'; password = 'wrong-x' } $null '127.0.0.1'
  Ok 'XFF 127.0.0.1 does not bypass' ($loop.code -eq 429 -or $loop.code -eq 423) ("code=" + $loop.code)

  StopServer
} finally {
  StopServer
  Pop-Location
}

$results | ForEach-Object { Write-Output $_ }
$fail = ($results | Where-Object { $_ -like 'FAIL*' }).Count
Write-Output ("---- {0} checks, {1} failed ----" -f $results.Count, $fail)
