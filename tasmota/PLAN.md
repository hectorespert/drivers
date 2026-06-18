# Plan: Soporte de Múltiples Salidas en el Driver Tasmota

## Contexto

Actualmente el driver Tasmota (`tasmota/http.go`) solo soporta **una salida por instancia**. 
El struct `httpDriver` implementa directamente las interfaces `hal.DigitalOutputPin` y `hal.PWMChannel`, 
lo que significa que el propio driver actúa como un único pin.

Para dispositivos con múltiples relés (ej: Sonoff 4CH, Sonoff Dual), 
se necesitaría crear múltiples instancias del driver con la misma IP, lo cual es ineficiente.

### Configuración actual

```json
{
  "Address": "192.168.1.46",
  "Output": "2"
}
```

- `Address` (string): IP del dispositivo Tasmota
- `Output` (integer, default: 0): Número de salida a controlar (Power0, Power1, Power2...)

### Comportamiento actual

- 1 pin por instancia de driver
- `pin.Name()` → `"Tasmota"`
- `pin.Number()` → `0`
- `Write(true)` → `GET /cm?cmnd=Power<output> true`
- `Set(50)` → `GET /cm?cmnd=Dimmer 50`
- `LastState()` → `GET /cm?cmnd=Power<output>` y parsea respuesta JSON

## Objetivo

Permitir que una sola instancia del driver gestione **N salidas** de un dispositivo Tasmota,
**añadiendo un nuevo parámetro `"Outputs"`** y manteniendo `"Output"` sin cambios para 
compatibilidad total hacia atrás.

## Nuevo parámetro `"Outputs"`

Se añade un parámetro **nuevo** `"Outputs"` (nótese la "s" final) que indica el número total
de salidas del dispositivo. El parámetro `"Output"` mantiene su significado original.

### Configuración nueva

```json
{
  "Address": "192.168.1.46",
  "Output": "2",
  "Outputs": "4"
}
```

- `Address` (string): IP del dispositivo Tasmota (sin cambios)
- `Output` (integer, default: 0): Número de salida a controlar en modo single-pin (sin cambios)
- `Outputs` (integer, default: 0): **NUEVO** - Número total de salidas. Si > 0, crea múltiples pins.

### Lógica de comportamiento

| "Outputs" | "Output" | Comportamiento |
|-----------|----------|---------------|
| `0` (default/ausente) | cualquier valor | **Modo legacy**: 1 pin en Power\<Output\> (idéntico al actual) |
| `2` | (ignorado) | 2 pins: Power1, Power2 |
| `4` | (ignorado) | 4 pins: Power1, Power2, Power3, Power4 |

### Compatibilidad total

- **Sin `"Outputs"`**: comportamiento 100% idéntico al actual. Ningún breaking change.
- **Con `"Outputs"` > 0**: modo multi-pin. El parámetro `"Output"` se ignora en este modo.
- Los tests existentes no requieren modificación alguna.

## Cambios Propuestos

### 1. Crear struct `tasmotaPin`

Extraer la lógica de pin (Write, Set, LastState, Name, Number, Close) a un struct separado:

- `tasmotaPin` contendrá: `address string`, `number int`
- Implementará: `hal.PWMChannel` (que incluye `hal.DigitalOutputPin` y `hal.Pin`)
- Cada pin se comunicará con `Power<number>` y `Dimmer<number>` del dispositivo
- `Name()` devolverá `"Tasmota"` si hay un solo pin, o `"Tasmota Pin <n>"` si hay varios
- `Number()` devolverá el número de output del pin

### 2. Refactorizar `httpDriver`

- Eliminar la implementación directa de interfaces de pin del driver
- Añadir campo `pins []*tasmotaPin`
- Los métodos `Pins()`, `DigitalOutputPins()`, `DigitalOutputPin(int)`, `PWMChannels()`, `PWMChannel(int)` 
  iterarán sobre el slice de pins

### 3. Lógica de creación en `NewDriver`

```
Si Outputs == 0 (o ausente):
    → Modo legacy: Crear 1 pin en Power<Output> (comportamiento actual)
Si Outputs > 0:
    → Modo multi-pin: Crear N pins: Power1, Power2, ..., PowerN
    → Se ignora el valor de "Output"
```

### 4. Nuevo parámetro en la factory

Añadir a `parameters`:
```go
{
    Name:    "Outputs",
    Type:    hal.Integer,
    Order:   2,
    Default: 0,
}
```

### 5. Validación de parámetros

- `"Address"`: obligatorio, string, longitud 1-255 (sin cambios)
- `"Output"`: opcional, integer >= 0, default 0 (sin cambios)
- `"Outputs"`: opcional, integer >= 0, default 0 (nuevo)

## Tests

### Tests existentes (sin cambios necesarios)

- `TestHttpDriver_AsDigitalOut`: Usa `"Output": "2"` sin `"Outputs"` → modo legacy, 1 pin. Sin cambios.
- `TestHttpDriver_AsPWMDriver`: Usa `"Output": "0"` → modo legacy, 1 pin. Sin cambios.
- `TestHttpDriver_FactoryValidateParameters`: Sin cambios.

### Tests nuevos a añadir

- `TestHttpDriver_MultipleOutputs`: Configurar con `"Outputs": "4"`, verificar 4 pins (Power1-Power4)
- `TestHttpDriver_MultipleOutputs_DigitalOutput`: Verificar interfaz DigitalOutputDriver con múltiples pins
- `TestHttpDriver_MultipleOutputs_PWM`: Verificar interfaz PWMDriver con múltiples pins
- `TestHttpDriver_PinBoundsCheck`: Verificar error al acceder a pin fuera de rango

## Referencia

Se sigue el patrón del driver `shelly/shelly25.go` que maneja 2 relés con un slice de `[]*Relay`.
