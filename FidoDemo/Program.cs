using Microsoft.AspNetCore.Mvc;
using System.Text.Json;
using System.IO;

var builder = WebApplication.CreateBuilder(args);

builder.WebHost.UseUrls("http://localhost:5050");

// CORS Probleminin qarşısını almaq üçün
builder.Services.AddCors(options => {
    options.AddDefaultPolicy(p => p.AllowAnyOrigin().AllowAnyHeader().AllowAnyMethod());
});

var app = builder.Build();
app.UseCors();

// ==========================================
// 1. FAYL BAZASI MƏNTİQİ (Server unutmasın)
// ==========================================
var dbPath = "devices.json";

// Server açılanda faylı oxuyur (Fayl yoxdursa, boş yaddaş yaradır)
var RegisteredDevices = File.Exists(dbPath) 
    ? JsonSerializer.Deserialize<Dictionary<string, string>>(File.ReadAllText(dbPath)) 
    : new Dictionary<string, string>();

// Dəyişiklik olanda fayla yazmaq üçün funksiya
void SaveDatabase() {
    File.WriteAllText(dbPath, JsonSerializer.Serialize(RegisteredDevices));
}

// ==========================================
// 2. HTML VƏ JAVASCRIPT (Sıfır-Klik Məntiqi)
// ==========================================
string GetHtml() => @"
<!DOCTYPE html>
<html>
<head>
    <meta charset='utf-8'>
    <title>Web Crypto Zero-Click</title>
    <style>
        body { font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; background: #f4f7f9; margin: 0; }
        .card { background: white; padding: 30px; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; width: 450px; }
        #status { font-size: 18px; font-weight: bold; margin-top: 20px; color: #555; }
        .log { margin-top:15px; font-size: 12px; color: gray; text-align: left; background: #eee; padding: 10px; border-radius: 5px; font-family: monospace; word-wrap: break-word; }
    </style>
</head>
<body>
    <div class='card'>
        <h2>Cihaz Tanıma (Sıfır Klik)</h2>
        <div id='status'>⏳ Sistem yoxlanılır...</div>
        <div class='log' id='log'></div>
    </div>

    <script>
        const DB_NAME = 'DeviceAuthDB';
        const KEY_ALIAS = 'device-private-key';

        function logInfo(msg) {
            document.getElementById('log').innerHTML += msg + '<br>';
            console.log(msg);
        }

        // Unikal Cihaz ID-si
        function getDeviceId() {
            let id = localStorage.getItem('unique-device-id');
            if (!id) {
                id = crypto.randomUUID ? crypto.randomUUID() : 'dev-' + Math.random().toString(36).substring(2);
                localStorage.setItem('unique-device-id', id);
            }
            return id;
        }

        async function getOrGenerateKey() {
            // Addım 1: Açar yoxlanışı
            let privateKey = await new Promise((resolve, reject) => {
                let request = indexedDB.open(DB_NAME, 1);
                request.onupgradeneeded = (e) => e.target.result.createObjectStore('keys');
                request.onsuccess = (e) => {
                    let db = e.target.result;
                    if (!db.objectStoreNames.contains('keys')) return resolve(null);
                    
                    let tx = db.transaction('keys', 'readonly');
                    let store = tx.objectStore('keys');
                    let getReq = store.get(KEY_ALIAS);
                    
                    getReq.onsuccess = () => resolve(getReq.result);
                    getReq.onerror = () => reject(new Error('Açar oxuna bilmədi'));
                };
                request.onerror = () => reject(new Error('IndexedDB açıla bilmədi'));
            });

            if (privateKey) {
                logInfo('Lokal açar tapıldı.');
                return privateKey;
            }

            // Addım 2: Açar yaradılması
            logInfo('Yeni açar yaradılır (Extractable: false)...');
            let keyPair = await window.crypto.subtle.generateKey(
                { name: 'RSASSA-PKCS1-v1_5', modulusLength: 2048, publicExponent: new Uint8Array([1, 0, 1]), hash: 'SHA-256' },
                false, 
                ['sign', 'verify']
            );

            // Addım 3: Yaddaşa yazma
            await new Promise((resolve, reject) => {
                let request = indexedDB.open(DB_NAME, 1);
                request.onsuccess = (e) => {
                    let db = e.target.result;
                    let tx = db.transaction('keys', 'readwrite');
                    let store = tx.objectStore('keys');
                    let putReq = store.put(keyPair.privateKey, KEY_ALIAS);
                    
                    putReq.onsuccess = () => resolve();
                    putReq.onerror = () => reject(new Error('Açar yazıla bilmədi'));
                };
            });
            
            // Addım 4: Qeydiyyat
            let pubKeyExported = await window.crypto.subtle.exportKey('spki', keyPair.publicKey);
            let pubKeyPem = btoa(String.fromCharCode(...new Uint8Array(pubKeyExported)));
            
            logInfo('Açar serverə göndərilir...');
            let regResp = await fetch('/api/register-device', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ deviceId: getDeviceId(), publicKey: pubKeyPem })
            });

            if (!regResp.ok) {
                let errMsg = await regResp.text();
                throw new Error(errMsg || 'Server qeydiyyatı rədd etdi.');
            }
            
            return keyPair.privateKey;
        }

        async function authenticate() {
            try {
                let privateKey = await getOrGenerateKey();
                
                logInfo('Təsdiq kodu alınır...');
                let challengeResp = await fetch('/api/get-challenge');
                if (!challengeResp.ok) throw new Error('Challenge alına bilmədi.');
                
                let { challenge } = await challengeResp.json();

                logInfo('Səssiz imzalama...');
                let encoder = new TextEncoder();
                let signature = await window.crypto.subtle.sign('RSASSA-PKCS1-v1_5', privateKey, encoder.encode(challenge));

                logInfo('İmza yoxlanılır...');
                let resp = await fetch('/api/verify-signature', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        deviceId: getDeviceId(),
                        signature: btoa(String.fromCharCode(...new Uint8Array(signature))),
                        challenge: challenge
                    })
                });

                let statusEl = document.getElementById('status');
                if (resp.ok) {
                    statusEl.innerText = '✅ Cihaz tanındı! Giriş uğurludur.';
                    statusEl.style.color = 'green';
                    logInfo('Proses bitdi!');
                } else {
                    statusEl.innerText = '❌ Cihaz rədd edildi!';
                    statusEl.style.color = 'red';
                }
            } catch (err) {
                document.getElementById('status').innerText = '❌ Giriş qadağandır!';
                document.getElementById('status').style.color = 'red';
                logInfo('<span style=""color:red"">XƏTA: ' + err.message + '</span>');
            }
        }

        window.onload = authenticate;
    </script>
</body>
</html>";

// ==========================================
// 3. API ENDPOINTLƏRİ
// ==========================================

app.MapGet("/", () => Results.Content(GetHtml(), "text/html"));

app.MapGet("/api/get-challenge", () => {
    return Results.Json(new { challenge = Guid.NewGuid().ToString() });
});

app.MapPost("/api/register-device", ([FromBody] RegisterRequest req) => {
    
    // KİLİD: Əgər bazada ən az 1 cihaz varsa və gələn cihaz fərqlidirsə, BLOKLA!
    if (RegisteredDevices.Count >= 1 && !RegisteredDevices.ContainsKey(req.DeviceId)) {
        Console.WriteLine($"[TƏHLÜKƏ] Kənar cihaz cəhdi BLOKLANDI! Cihaz ID: {req.DeviceId}");
        return Results.BadRequest("Sistem artıq başqa bir cihaza lehimlənib! Yeni cihaz qəbul edilmir.");
    }

    // Əgər ilk cihazdırsa, qeydə al və FAYLA YAZ
    if (!RegisteredDevices.ContainsKey(req.DeviceId)) {
        RegisteredDevices[req.DeviceId] = req.PublicKey;
        SaveDatabase(); // <-- Yaddaş itkisinin qarşısını alan sehrli kod
        Console.WriteLine($"[QEYDİYYAT] İlk cihaz lehimləndi və devices.json faylına yazıldı. Sistem kilidləndi!");
    }
    
    return Results.Ok();
});

app.MapPost("/api/verify-signature", ([FromBody] VerifyRequest req) => {
    if (!RegisteredDevices.TryGetValue(req.DeviceId, out var publicKeyPem)) {
        Console.WriteLine($"[XƏBƏRDARLIQ] Naməlum cihazın imzası rədd edildi!");
        return Results.Unauthorized();
    }
    
    Console.WriteLine($"[GİRİŞ UĞURLU] Cihaz tanındı: {req.DeviceId}");
    return Results.Ok();
});

app.Run();

// ==========================================
// 4. MODEL TƏYİNATLARI (Ən sonda olmalıdır)
// ==========================================
public record RegisterRequest(string DeviceId, string PublicKey);
public record VerifyRequest(string DeviceId, string Signature, string Challenge);