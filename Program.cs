using Fido2NetLib;
using Fido2NetLib.Objects;
using Microsoft.AspNetCore.Mvc;
using System.Text;
using System.Text.Json;

var builder = WebApplication.CreateBuilder(args);

// SİZİN GİTHUB CODESPACES LİNKİNİZƏ UYĞUNLAŞDIRILMIŞ FIDO2 AYARLARI
builder.Services.AddFido2(options =>
{
    // Domain hissəsi (https:// və ən sondakı / işarəsi OLMADAN)
    options.ServerDomain = "zany-journey-j95665w6569f557j-5287.app.github.dev"; 
    options.ServerName = "Rəşadın Təhlükəsiz Saytı";
    
    // Origin hissəsi (https:// İLƏ BİRLİKDƏ, amma sondakı / işarəsi OLMADAN)
    options.Origins = new HashSet<string> { "https://zany-journey-j95665w6569f557j-5287.app.github.dev" }; 
    options.TimestampDriftTolerance = 300000;
});

var app = builder.Build();

// Müvəqqəti Yaddaş
CredentialCreateOptions? tempOptions = null;
byte[]? savedCredentialId = null;

// 1. Əsas Səhifə (HTML)
app.MapGet("/", () => Results.Content(GetHtml(), "text/html"));

// 2. Cihazı tanımaq üçün ilkin sorğu (Açar yaradılma tələbi)
app.MapPost("/register-options", (IFido2 fido2) =>
{
    var user = new Fido2User { DisplayName = "Rəşad", Name = "resad", Id = Encoding.UTF8.GetBytes("resad_123") };
    
    var authSelection = new AuthenticatorSelection
    {
        AuthenticatorAttachment = AuthenticatorAttachment.Platform, // Yalnız bu kompüterin öz TPM-i
        UserVerification = UserVerificationRequirement.Discouraged // PİN və barmaq izi TƏLƏB ETMƏ! (0 klikə ən yaxın)
    };

    tempOptions = fido2.RequestNewCredential(user, new List<PublicKeyCredentialDescriptor>(), authSelection, AttestationConveyancePreference.None, new AuthenticationExtensionsClientInputs());
    
    return Results.Json(tempOptions);
});

// 3. Yaradılmış açarı təsdiqləyib bazaya (yaddaşa) yazan endpoint
app.MapPost("/register", async (IFido2 fido2, [FromBody] JsonElement response) =>
{
    try
    {
        var attestationResponse = JsonSerializer.Deserialize<AuthenticatorAttestationRawResponse>(response.GetRawText());
        IsCredentialIdUniqueToUserAsyncDelegate callback = async (args, ct) => true;
        
        var success = await fido2.MakeNewCredentialAsync(attestationResponse, tempOptions, callback);
        
        savedCredentialId = success.Result.CredentialId; // Açar qeydə alındı
        return Results.Ok(new { message = "UĞURLU: Kompüteriniz (TPM) sistemə tanıdıldı və kilidləndi!" });
    }
    catch (Exception ex)
    {
        return Results.BadRequest(new { message = ex.Message });
    }
});

app.Run();

// ----- HTML VƏ JAVASCRIPT KODU (Ön Plan) -----
static string GetHtml() => @"
<!DOCTYPE html>
<html>
<head>
    <meta charset='utf-8'>
    <title>Cihaz Kilidləmə Testi</title>
    <style>
        body { font-family: Arial; display:flex; justify-content:center; align-items:center; height: 100vh; background: #f0f2f5; margin:0;}
        .box { text-align: center; background: white; padding: 40px; border-radius: 10px; box-shadow: 0 4px 10px rgba(0,0,0,0.1); }
        button { padding: 15px 30px; font-size: 18px; cursor: pointer; background: #007bff; color: white; border: none; border-radius: 5px; transition: 0.3s;}
        button:hover { background: #0056b3; }
        #status { margin-top: 20px; font-weight: bold; font-size: 16px; }
    </style>
</head>
<body>
    <div class='box'>
        <h2>Bu cihazı sistemə bağla</h2>
        <p>Təsdiq pəncərəsi açılanda PİN istəməyəcək, sadəcə ""Davam et/OK"" basın.</p>
        <button onclick='register()'>Cihazı Qeydiyyata Al (Test et)</button>
        <p id='status' style='color: gray;'>Gözlənilir...</p>
    </div>

    <script>
        async function register() {
            let statusEl = document.getElementById('status');
            statusEl.innerText = 'İşləyir... Lütfən pəncərədə təsdiqləyin.';
            statusEl.style.color = 'orange';

            try {
                // 1. Serverdən icazə və parametrləri al
                let resp = await fetch('/register-options', { method: 'POST' });
                let options = await resp.json();

                // Brauzer üçün formatlama
                options.challenge = base64ToArray(options.challenge);
                options.user.id = base64ToArray(options.user.id);

                // 2. Windows Hello / TPM-ə səssiz müraciət (PİN istəməyəcək)
                let cred = await navigator.credentials.create({ publicKey: options });

                // 3. Yaradılan Açarı serverə göndər
                let attestation = {
                    id: cred.id,
                    rawId: arrayToBase64(cred.rawId),
                    type: cred.type,
                    response: {
                        attestationObject: arrayToBase64(cred.response.attestationObject),
                        clientDataJSON: arrayToBase64(cred.response.clientDataJSON)
                    }
                };

                let finalResp = await fetch('/register', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(attestation)
                });

                let result = await finalResp.json();
                
                if(finalResp.ok) {
                    statusEl.innerText = result.message;
                    statusEl.style.color = 'green';
                } else {
                    statusEl.innerText = 'Xəta: ' + result.message;
                    statusEl.style.color = 'red';
                }

            } catch (err) {
                statusEl.innerText = 'İmtina edildi və ya Xəta: ' + err.message;
                statusEl.style.color = 'red';
            }
        }

        // Base64 <-> ArrayBuffer köməkçi funksiyaları
        function base64ToArray(base64) {
            let binary = window.atob(base64.replace(/-/g, '+').replace(/_/g, '/'));
            let bytes = new Uint8Array(binary.length);
            for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
            return bytes;
        }
        function arrayToBase64(buffer) {
            let binary = '';
            let bytes = new Uint8Array(buffer);
            for (let i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i]);
            return window.btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
        }
    </script>
</body>
</html>";