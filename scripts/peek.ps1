$Base = 'http://127.0.0.1:8000'
function Req($method, $uri, $body, $token) {
  $p = @{ Uri = $uri; Method = $method; UseBasicParsing = $true; TimeoutSec = 15; ErrorAction = 'Stop' }
  if ($body) { $p.Body = ($body | ConvertTo-Json -Depth 6); $p.ContentType = 'application/json; charset=utf-8' }
  if ($token) { $p.Headers = @{ Authorization = "Bearer $token" } }
  try { $resp = Invoke-WebRequest @p; return @{ StatusCode = [int]$resp.StatusCode; Content = [string]$resp.Content } } catch { return @{ StatusCode = -1; Content = '' } }
}
$t = ((Req 'POST' "$Base/api/auth/login" @{ username='admin'; password='admin123' }).Content | ConvertFrom-Json).data.access_token
$cards = ((Req 'GET' "$Base/api/card-records?size=5" $null $t).Content | ConvertFrom-Json).data.items
Write-Output "recent card codes:"
$cards | ForEach-Object { Write-Output ("  " + $_.card_code + "  " + $_.call_sign + "  " + $_.flow_status) }
$mail = (Req 'GET' "$Base/api/public/station-mail-info" $null $null).Content | ConvertFrom-Json
Write-Output ("station-mail-info: " + $mail.data.PSObject.Properties.Name -join ',')