using Fido2NetLib;
using Fido2NetLib.Objects;
using Microsoft.AspNetCore.Mvc;
using System.Text;
using System.Text.Json;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddFido2(options =>
{
    options.ServerDomain = "http://localhost:5050";
    options.ServerName = "Rəşadın Təhlükəsiz Saytı";
    options.Origins = new HashSet<string> { "http://localhost:5050" };
});

var app = builder.Build();

CredentialCreateOptions? tempOptions = null;
byte[]? savedCredentialId = null;

app.MapGet("/", () => Results.Content(GetHtml(), "text/html"));

app.MapPost("/register-options", (IFido2 fido2) =>
{
    var user = new Fido2User
    {
        DisplayName = "Rəşad",
        Name = "resad",
        Id = Encoding.UTF8.GetBytes("resad_123")
    };

    var authSelection = new AuthenticatorSelection
    {
        AuthenticatorAttachment = AuthenticatorAttachment.Platform,
        UserVerification = UserVerificationRequirement.Discouraged
    };

    tempOptions = fido2.RequestNewCredential(
        user,
        new List<PublicKeyCredentialDescriptor>(),
        authSelection,
        AttestationConveyancePreference.None,
        new AuthenticationExtensionsClientInputs()
    );

    return Results.Json(tempOptions);
});

app.MapPost("/register", async (IFido2 fido2, [FromBody] JsonElement response) =>
{
    try
    {
        var attestationResponse = JsonSerializer.Deserialize<AuthenticatorAttestationRawResponse>(response.GetRawText());

        if (attestationResponse == null || tempOptions == null)
            return Results.BadRequest(new { message = "Məlumatlar tam deyil." });

        IsCredentialIdUniqueToUserAsyncDelegate callback = async (args, ct) => true;

        var success = await fido2.MakeNewCredentialAsync(attestationResponse, tempOptions, callback);

        if (success.Result == null)
            return Results.BadRequest(new { message = "Credential yaradıla bilmədi." });

        savedCredentialId = success.Result.CredentialId;
        return Results.Ok(new { message = "UĞURLU: Kompüteriniz (TPM) sistemə tanıdıldı və kilidləndi!" });
    }
    catch (Exception ex)
    {
        return Results.BadRequest(new { message = ex.Message });
    }
});

app.Run();

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
                let resp = await fetch('/register-options', { method: 'POST' });
                let options = await resp.json();
                options.challenge = base64ToArray(options.challenge);
                options.user.id = base64ToArray(options.user.id);
                let cred = await navigator.credentials.create({ publicKey: options });
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