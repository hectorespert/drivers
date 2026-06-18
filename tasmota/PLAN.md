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
**sin añadir nuevos parámetros de configuración**, reinterpretando el parámetro `"Output"` existente.

## Redefinición del parámetro `"Output"`

Se reinterpreta `"Output"` como **número de salidas** (cantidad), en lugar de "cuál salida usar":

| Valor de "Output" | Comportamiento |
|-------------------|---------------|
| `0` (default) | 1 pin: Power0 (comportamiento legacy idéntico) |
| `1` | 1 pin: Power1 |
| `2` | 2 pins: Power1, Power2 |
| `4` | 4 pins: Power1, Power2, Power3, Power4 |

### Compatibilidad

- `"Output": "0"` o ausente → **Idéntico al actual**: 1 pin en Power0
- `"Output": "1"` → **Idéntico al actual**: 1 pin en Power1
- `"Output": ">1"` → **Cambio de comportamiento**: antes controlaba 1 salida específica, ahora crea N salidas

> **Nota**: Los usuarios que usaban `"Output": "2"` para controlar *específicamente* Power2
> ahora obtendrán 2 pins (Power1 y Power2). Esto es un breaking change para ese caso,
> pero se considera aceptable dado que el uso principal de Output > 1 en dispositivos
> multi-relé es precisamente controlar todas las salidas disponibles.

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
Si Output == 0:
    → Crear 1 pin en Power0 (legacy)
Si Output == 1:
    → Crear 1 pin en Power1
Si Output > 1:
    → Crear N pins: Power1, Power2, ..., PowerN
```

### 4. Validación de parámetros

- `"Address"`: obligatorio, string, longitud 1-255 (sin cambios)
- `"Output"`: opcional, integer >= 0, default 0 (sin cambios en tipo ni validación)

## Tests

### Tests existentes (deben actualizarse mínimamente)

- `TestHttpDriver_AsDigitalOut`: Usaba `"Output": "2"` → ahora producirá 2 pins. Actualizar assertions.
- `TestHttpDriver_AsPWMDriver`: Usaba `"Output": "0"` → sigue igual (1 pin, Power0)
- `TestHttpDriver_FactoryValidateParameters`: Sin cambios en validación

### Tests nuevos a añadir

- `TestHttpDriver_MultipleOutputs`: Configurar con `"Output": "4"`, verificar 4 pins
- `TestHttpDriver_SingleOutput`: Verificar que `"Output": "1"` crea 1 pin en Power1
- `TestHttpDriver_PinBoundsCheck`: Verificar error al acceder a pin fuera de rango

## Correcciones adicionales (opcionales, en commit separado)

- Corregir el default de Address: `"192.1.168.4"` → `"192.168.1.4"`
- El comando Dimmer en modo múltiple usará `Dimmer<n>` en lugar de solo `Dimmer`

## Referencia

Se sigue el patrón del driver `shelly/shelly25.go` que maneja 2 relés con un slice de `[]*Relay`.


