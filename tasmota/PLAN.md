# Plan: Soporte de Múltiples Salidas en el Driver Tasmota

## Contexto

Actualmente el driver Tasmota (`tasmota/http.go`) solo soporta **una salida por instancia**. 
El struct `httpDriver` implementa directamente las interfaces `hal.DigitalOutputPin` y `hal.PWMChannel`, 
lo que significa que el propio driver actúa como un único pin.

Para dispositivos con múltiples relés (ej: Sonoff 4CH, Sonoff Dual), 
se necesitaría crear múltiples instancias del driver con la misma IP, lo cual es ineficiente.

## Objetivo

Permitir que una sola instancia del driver gestione **N salidas** de un dispositivo Tasmota,
similar a como el driver Shelly25 maneja 2 relés.

## Cambios Propuestos

### 1. Crear struct `tasmotaPin`

Extraer la lógica de pin (Write, Set, LastState, Name, Number, Close) a un struct separado:

- `tasmotaPin` contendrá: `address string`, `number int`
- Implementará: `hal.PWMChannel` (que incluye `hal.DigitalOutputPin` y `hal.Pin`)
- Cada pin se comunicará con `Power<number>` y `Dimmer<number>` del dispositivo

### 2. Refactorizar `httpDriver`

- Eliminar los campos `output int` y los métodos de pin del driver
- Añadir campo `pins []*tasmotaPin`
- Los métodos `Pins()`, `DigitalOutputPins()`, `DigitalOutputPin(int)`, `PWMChannels()`, `PWMChannel(int)` 
  iterarán sobre el slice de pins

### 3. Cambiar parámetro de configuración

- Reemplazar el parámetro `"Output"` (número de salida) por `"Outputs"` (cantidad de salidas)
- Valor por defecto: `1` (retrocompatible con dispositivos de un solo relé)
- Validación: debe ser >= 1

### 4. Actualizar la factory (`NewDriver`)

- Crear N pins con numeración desde 1 (Power1, Power2, ... PowerN)
- Esto coincide con la convención de Tasmota

### 5. Correcciones adicionales (opcionales)

- Corregir el default de Address: `"192.1.168.4"` → `"192.168.1.4"`
- El comando Dimmer deberá incluir el canal: `Dimmer<n>` en lugar de solo `Dimmer`

## Impacto en la API

| Antes | Después |
|-------|---------|
| `"Output": "2"` (usa la salida 2) | `"Outputs": "2"` (crea 2 salidas: pin 1 y pin 2) |
| 1 pin por driver | N pins por driver |
| `pin.Name()` → `"Tasmota"` | `pin.Name()` → `"Tasmota Pin 1"`, `"Tasmota Pin 2"`, ... |
| `pin.Number()` → `0` | `pin.Number()` → `1`, `2`, ... |

## Tests a Actualizar

- `TestHttpDriver_AsDigitalOut`: Configurar con 2 outputs, verificar 2 pins
- `TestHttpDriver_AsPWMDriver`: Verificar acceso a PWM channels por índice
- `TestHttpDriver_FactoryValidateParameters`: Usar nuevo parámetro `"Outputs"`
- Añadir test para validar error en pin fuera de rango
- Añadir test para valor por defecto (1 output cuando no se especifica)

## Referencia

Se sigue el patrón del driver `shelly/shelly25.go` que maneja 2 relés con un slice de `[]*Relay`.
