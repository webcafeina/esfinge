-- El índice global. Lo de cada cuenta vive en su Durable Object; aquí solo lo
-- que hay que buscar sin saber aún de qué cuenta se trata.

-- Correo → cuenta. El correo va normalizado: minúsculas y sin espacios.
CREATE TABLE cuentas (
	correo TEXT PRIMARY KEY,
	cuenta TEXT NOT NULL UNIQUE,
	creada INTEGER NOT NULL
);

-- Altas a medio hacer: el código que se mandó a un correo que aún no tiene cuenta.
CREATE TABLE altas (
	correo TEXT PRIMARY KEY,
	codigo TEXT NOT NULL,
	caduca INTEGER NOT NULL,
	intentos INTEGER NOT NULL DEFAULT 0
);

-- Cuántos códigos de alta se han mandado a cada correo, para no bombardear un buzón.
CREATE TABLE envios_alta (
	correo TEXT NOT NULL,
	momento INTEGER NOT NULL
);
CREATE INDEX envios_alta_correo ON envios_alta (correo, momento);

-- Contadores por día. La clave nunca es una IP en claro: es un HMAC de ella.
CREATE TABLE contadores (
	clave TEXT NOT NULL,
	dia TEXT NOT NULL,
	n INTEGER NOT NULL,
	PRIMARY KEY (clave, dia)
);

-- Quién puede darse de alta mientras el registro es por invitación: un correo
-- entero («ana@ejemplo.com») o un dominio («@webcafeina.com»).
CREATE TABLE admision (
	patron TEXT PRIMARY KEY
);

-- El buzón del Worker de pruebas. En producción esta tabla existe y está vacía:
-- el cartero de verdad es Resend, y la ruta que la lee da 404.
CREATE TABLE buzon_pruebas (
	correo TEXT NOT NULL,
	asunto TEXT NOT NULL,
	cuerpo TEXT NOT NULL,
	momento INTEGER NOT NULL
);
