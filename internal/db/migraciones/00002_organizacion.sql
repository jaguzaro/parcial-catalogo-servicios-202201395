-- +goose Up
CREATE TABLE empresa (
    id             bigint GENERATED ALWAYS AS IDENTITY,
    codigo         text        NOT NULL,
    nombre         text        NOT NULL,
    activo         boolean     NOT NULL DEFAULT true,
    creado_en      timestamptz NOT NULL DEFAULT now(),
    actualizado_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT empresa_pk PRIMARY KEY (id),
    CONSTRAINT empresa_codigo_uk UNIQUE (codigo),
    CONSTRAINT empresa_codigo_ck CHECK (btrim(codigo) <> '' AND codigo = btrim(codigo)),
    CONSTRAINT empresa_nombre_ck CHECK (btrim(nombre) <> '')
);

CREATE TABLE area (
    id             bigint GENERATED ALWAYS AS IDENTITY,
    empresa_id     bigint      NOT NULL,
    codigo         text        NOT NULL,
    nombre         text        NOT NULL,
    activo         boolean     NOT NULL DEFAULT true,
    creado_en      timestamptz NOT NULL DEFAULT now(),
    actualizado_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT area_pk PRIMARY KEY (id),
    CONSTRAINT area_empresa_fk FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT area_codigo_uk UNIQUE (empresa_id, codigo),
    CONSTRAINT area_codigo_ck CHECK (btrim(codigo) <> '' AND codigo = btrim(codigo)),
    CONSTRAINT area_nombre_ck CHECK (btrim(nombre) <> '')
);

CREATE TABLE departamento (
    id             bigint GENERATED ALWAYS AS IDENTITY,
    area_id        bigint      NOT NULL,
    codigo         text        NOT NULL,
    nombre         text        NOT NULL,
    activo         boolean     NOT NULL DEFAULT true,
    creado_en      timestamptz NOT NULL DEFAULT now(),
    actualizado_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT departamento_pk PRIMARY KEY (id),
    CONSTRAINT departamento_area_fk FOREIGN KEY (area_id) REFERENCES area (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT departamento_codigo_uk UNIQUE (area_id, codigo),
    CONSTRAINT departamento_codigo_ck CHECK (btrim(codigo) <> '' AND codigo = btrim(codigo)),
    CONSTRAINT departamento_nombre_ck CHECK (btrim(nombre) <> '')
);

CREATE TABLE seccion (
    id              bigint GENERATED ALWAYS AS IDENTITY,
    departamento_id bigint      NOT NULL,
    codigo          text        NOT NULL,
    nombre          text        NOT NULL,
    activo          boolean     NOT NULL DEFAULT true,
    creado_en       timestamptz NOT NULL DEFAULT now(),
    actualizado_en  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT seccion_pk PRIMARY KEY (id),
    CONSTRAINT seccion_departamento_fk FOREIGN KEY (departamento_id) REFERENCES departamento (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT seccion_codigo_uk UNIQUE (departamento_id, codigo),
    CONSTRAINT seccion_codigo_ck CHECK (btrim(codigo) <> '' AND codigo = btrim(codigo)),
    CONSTRAINT seccion_nombre_ck CHECK (btrim(nombre) <> '')
);

CREATE TABLE puesto (
    id             bigint GENERATED ALWAYS AS IDENTITY,
    seccion_id     bigint      NOT NULL,
    codigo         text        NOT NULL,
    nombre         text        NOT NULL,
    activo         boolean     NOT NULL DEFAULT true,
    creado_en      timestamptz NOT NULL DEFAULT now(),
    actualizado_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT puesto_pk PRIMARY KEY (id),
    CONSTRAINT puesto_seccion_fk FOREIGN KEY (seccion_id) REFERENCES seccion (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT puesto_codigo_uk UNIQUE (seccion_id, codigo),
    CONSTRAINT puesto_codigo_ck CHECK (btrim(codigo) <> '' AND codigo = btrim(codigo)),
    CONSTRAINT puesto_nombre_ck CHECK (btrim(nombre) <> '')
);

CREATE TABLE usuario (
    id              bigint GENERATED ALWAYS AS IDENTITY,
    puesto_id       bigint      NOT NULL,
    nombre          text        NOT NULL,
    usuario         text        NOT NULL,
    correo          text,
    hash_contrasena text        NOT NULL,
    rol             text        NOT NULL,
    activo          boolean     NOT NULL DEFAULT true,
    creado_en       timestamptz NOT NULL DEFAULT now(),
    actualizado_en  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT usuario_pk PRIMARY KEY (id),
    CONSTRAINT usuario_puesto_fk FOREIGN KEY (puesto_id) REFERENCES puesto (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT usuario_nombre_ck CHECK (btrim(nombre) <> ''),
    CONSTRAINT usuario_usuario_ck CHECK (btrim(usuario) <> '' AND usuario = btrim(usuario) AND position('@' in usuario) = 0),
    CONSTRAINT usuario_correo_ck CHECK (correo IS NULL OR position('@' in correo) > 1),
    CONSTRAINT usuario_hash_ck CHECK (hash_contrasena LIKE '$argon2id$%'),
    CONSTRAINT usuario_rol_ck CHECK (rol IN ('administrador', 'consulta'))
);
CREATE UNIQUE INDEX usuario_usuario_uk ON usuario (lower(usuario));
CREATE UNIQUE INDEX usuario_correo_uk ON usuario (lower(correo)) WHERE correo IS NOT NULL;
CREATE INDEX usuario_puesto_idx ON usuario (puesto_id);

CREATE TABLE sesion (
    id          bigint GENERATED ALWAYS AS IDENTITY,
    usuario_id  bigint      NOT NULL,
    token_hash  bytea       NOT NULL,
    creada_en   timestamptz NOT NULL DEFAULT now(),
    expira_en   timestamptz NOT NULL,
    revocada_en timestamptz,
    CONSTRAINT sesion_pk PRIMARY KEY (id),
    CONSTRAINT sesion_usuario_fk FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT sesion_token_uk UNIQUE (token_hash),
    CONSTRAINT sesion_token_ck CHECK (octet_length(token_hash) = 32),
    CONSTRAINT sesion_expira_ck CHECK (expira_en > creada_en)
);
CREATE INDEX sesion_usuario_idx ON sesion (usuario_id);

-- +goose Down
DROP TABLE sesion;
DROP TABLE usuario;
DROP TABLE puesto;
DROP TABLE seccion;
DROP TABLE departamento;
DROP TABLE area;
DROP TABLE empresa;
