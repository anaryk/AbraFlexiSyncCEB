
# Go Application for File Processing and Payment Order Management

Tento program je navržen pro zpracování souborů a správu příkazů k úhradě. Je napsán v jazyce Go a využívá různé knihovny pro zpracování konfigurace, HTTP požadavků a logování.

## Funkcionalita

1. **Zpracování souborů**: Program prochází zadané adresáře a zpracovává soubory s příponou `.GPC` nebo `.gpc`. Tyto soubory jsou nahrány na server AbraCloud přes HTTP POST požadavek.
2. **Správa příkazů k úhradě**: Program kontroluje nové příkazy k úhradě, stahuje je a ukládá do určeného adresáře.
3. **Logování**: Program podporuje logování do souboru nebo do konzole.

## Konfigurace

### Konfigurační soubory

Program vyžaduje dva typy konfiguračních souborů ve formátu YAML:

1. **Central Configuration (central_config.yaml)**:
   ```yaml
   directories:
     - "/path/to/directory1"
     - "/path/to/directory2"
   payment_order_directories:
     - "/path/to/payment/order/directory1"
     - "/path/to/payment/order/directory2"
   log_to_file: true
   ```

2. **Directory Specific Configuration (config.yaml)**:
   ```yaml
   url: "http://example.com/api"
   username: "your_username"
   password: "your_password"
   ```

### Environment Variables

Alternativně můžete nastavit adresáře pomocí proměnných prostředí:
- `DIR_1`, `DIR_2`, ..., `DIR_N` pro hlavní adresáře.
- `PAYMENT_ORDER_DIR_1`, `PAYMENT_ORDER_DIR_2`, ..., `PAYMENT_ORDER_DIR_N` pro adresáře příkazů k úhradě.

## Jak spustit

1. Vytvořte potřebné konfigurační soubory.
2. Nastavte proměnné prostředí, pokud nepoužíváte centrální konfiguraci.
3. Spusťte program:

   ```sh
   go run main.go
   ```

## Závislosti

Program využívá následující knihovny:
- [zerolog](https://github.com/rs/zerolog) pro logování
- [yaml.v2](https://gopkg.in/yaml.v2) pro zpracování YAML souborů
- [encoding/json](https://pkg.go.dev/encoding/json) pro práci s JSON
- [encoding/xml](https://pkg.go.dev/encoding/xml) pro práci s XML

## Popis kódu

### Hlavní funkce

- **main**: Inicializuje logování, načítá centrální konfiguraci a spouští periodické zpracování adresářů.
- **processDirectories**: Zpracovává dané adresáře a volá funkce pro zpracování souborů a příkazů k úhradě.
- **processFiles**: Prochází soubory v adresáři, které mají příponu `.GPC` nebo `.gpc`, a nahrává je na server.
- **processPaymentOrders**: Kontroluje nové příkazy k úhradě a stahuje je do určeného adresáře.

### Načítání konfigurace

- **loadConfig**: Načítá specifickou konfiguraci adresáře z `config.yaml`.
- **loadCentralConfig**: Načítá centrální konfiguraci z `central_config.yaml`.

## Logování

Program podporuje logování do souboru, pokud je v centrální konfiguraci `log_to_file` nastaveno na `true`. Jinak se loguje do konzole.

## Příklady použití

- **Zpracování souborů**:
  ```sh
  go run main.go
  ```

- **Kontrola příkazů k úhradě**:
  Program pravidelně kontroluje nové příkazy k úhradě a ukládá je do určeného adresáře.

## Kontakt

Pro více informací nebo pomoc s tímto programem, prosím kontaktujte [vaše kontaktní informace].

Doufám, že vám tento README pomůže lépe pochopit, jak tento program funguje a jak jej použít. Pokud máte jakékoli dotazy nebo potřebujete další pomoc, neváhejte mě kontaktovat.
