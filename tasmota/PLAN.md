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
**manteniendo compatibilidad total** con la configuración y comportamiento existentes.

## Requisitos de Compatibilidad

1. El parámetro `"Output"` debe seguir funcionándose como antes
2. Cuando solo se pasa `"Output"`, el driver se comporta exactamente igual que antes (1 pin)
3. Los nombres de pin y numeración deben ser consistentes con el uso actual
4. Las URLs de la API Tasmota generadas deben ser idénticas al comportamiento actual

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
- `DigitalOutputPin(n)` y `PWMChannel(n)` acceden por **índice** del slice (0-based)

### 3. Añadir parámetro `"Outputs"` (nuevo, opcional)

- Añadir un **nuevo** parámetro `"Outputs"` (integer, default: 0)
- Semántica: número total de salidas a crear (desde Power1 hasta PowerN)
- `"Outputs": 0` o ausente → modo legacy, usa `"Output"` como antes

### 4. Lógica de creación en `NewDriver`

```
Si "Outputs" > 0:
    → Crear N pins: pin[0]=Power1, pin[1]=Power2, ..., pin[N-1]=PowerN
    → Ignorar "Output"
Si no:
    → Modo legacy: crear 1 solo pin con el número indicado en "Output"
    → Comportamiento idéntico al actual
```

### 5. Validación de parámetros

- `"Address"`: obligatorio, string, longitud 1-255 (sin cambios)
- `"Output"`: opcional, integer >= 0, default 0 (sin cambios)
- `"Outputs"`: opcional, integer >= 0, default 0 (nuevo)
- Si `"Outputs" > 0` y `"Output" > 0` simultáneamente: usar `"Outputs"` (prioridad al nuevo)

## Impacto en la API

### Modo Legacy (retrocompatible, sin cambios funcionales)

| Configuración | Comportamiento |
|--------------|----------------|
| `{"Address": "192.168.1.46", "Output": "2"}` | 1 pin, Power2, idéntico al actual |
| `{"Address": "192.168.1.46"}` | 1 pin, Power0, idéntico al actual |

### Modo Múltiples Salidas (nuevo)

| Configuración | Comportamiento |
|--------------|----------------|
| `{"Address": "192.168.1.46", "Outputs": "4"}` | 4 pins: Power1, Power2, Power3, Power4 |
| `{"Address": "192.168.1.46", "Outputs": "1"}` | 1 pin: Power1 |

## Tests

### Tests existentes (no deben romperse)

- `TestHttpDriver_AsDigitalOut`: Mantener con `"Output": "2"` → 1 pin, comportamiento legacy
- `TestHttpDriver_AsPWMDriver`: Mantener con `"Output": "0"` → 1 pin, comportamiento legacy
- `TestHttpDriver_FactoryValidateParameters`: Mantener validaciones de `"Address"`

### Tests nuevos a añadir

- `TestHttpDriver_MultipleOutputs`: Configurar con `"Outputs": "4"`, verificar 4 pins
- `TestHttpDriver_OutputsOverridesOutput`: Verificar que `"Outputs"` tiene prioridad sobre `"Output"`
- `TestHttpDriver_PinBoundsCheck`: Verificar error al acceder a pin fuera de rango
- `TestHttpDriver_LegacyCompatibility`: Verificar que sin `"Outputs"` funciona exactamente igual

## Correcciones adicionales (opcionales, en commit separado)

- Corregir el default de Address: `"192.1.168.4"` → `"192.168.1.4"`
- El comando Dimmer en modo múltiple usará `Dimmer<n>` en lugar de solo `Dimmer`

## Referencia

Se sigue el patrón del driver `shelly/shelly25.go` que maneja 2 relés con un slice de `[]*Relay`.

