# quick diagnostics for E2E failures
$Base = 'http://127.0.0.1:8000'

function Req($method, $uri, $body, $token) {
  $p = @{ Uri = $uri; Method = $method; UseBasicParsing = $true; TimeoutSec = 20; ErrorAction = 'Stop' }
  if ($body) { $p.Body = ($body | ConvertTo-Json -Depth 6); $p.ContentType = 'application/json; charset=utf-8' }
  if ($token) { $p.Headers = @{ Authorization = "Bearer $token" } }
  try {
    $resp = Invoke-WebRequest @p
    return @{ StatusCode = [int]$resp.StatusCode; Content = [string]$resp.Content }
  } catch {
    $sc = -1
    try { $sc = [int]$_.Exception.Response.StatusCode } catch { $sc = -1 }
    return @{ StatusCode = $sc; Content = '' }
  }
}

$r = Req 'POST' "$Base/api/auth/login" @{ username = 'admin'; password = 'admin123' }
$env = $r.Content | ConvertFrom-Json
$tok = $env.data.access_token
Write-Output ("login status=" + $r.StatusCode + "  tokenLen=" + $tok.Length + "  tokHead=" + $tok.Substring(0, [Math]::Min(20, $tok.Length)))

$r = Req 'GET' "$Base/api/exchange/online/requests?size=5" $null $tok
Write-Output ("list tokened status=" + $r.StatusCode)
Write-Output ("list body head: " + $r.Content.Substring(0, [Math]::Min(400, $r.Content.Length)))

$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'QSO'; call_sign = 'E2EQSO'; qso_date = '2024-11-05'; qso_freq = '14.270'; qso_mode = 'SSB'
  email = 'e2e@example.com'; address = 'x'
}
Write-Output ("duplicate submit status=" + $r.StatusCode + "  body: " + $r.Content.Substring(0, [Math]::Min(300, $r.Content.Length)))
