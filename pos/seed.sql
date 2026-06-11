-- Datos de prueba para desarrollo local
-- Productos típicos de abarrotes

INSERT OR IGNORE INTO DEPARTAMENTOS (id, nombre) VALUES
(1, 'Abarrotes'),
(2, 'Bebidas'),
(3, 'Lácteos'),
(4, 'Panadería'),
(5, 'Cuidado Personal'),
(6, 'Limpieza'),
(7, 'Botanas'),
(8, 'Carnicería');

INSERT OR IGNORE INTO MEDIDAS (codigo, nombre) VALUES
(1, 'Pieza'),
(2, 'Kilogramo'),
(3, 'Litro'),
(4, 'Paquete'),
(5, 'Docena');

INSERT OR IGNORE INTO PROV (num, nombre) VALUES
(1, 'Distribuidora Local'),
(2, 'Coca-Cola FEMSA'),
(3, 'Bimbo'),
(4, 'Grupo Modelo');

INSERT OR IGNORE INTO PRODUCTOS (codigo, descripcion, tventa, pcosto, pventa, dept, umedida, dinventario) VALUES
('75010001', 'Coca-Cola 600ml',      'p', 12.50, 17.00, 2, 3, 50),
('75010002', 'Coca-Cola 2L',         'p', 22.00, 30.00, 2, 3, 30),
('75010003', 'Pepsi 600ml',          'p', 11.00, 15.00, 2, 3, 40),
('75010004', 'Agua Bonafont 1L',     'p', 9.00,  13.00, 2, 3, 60),
('75010005', 'Jugo del Valle 1L',    'p', 18.00, 25.00, 2, 3, 25),
('75020001', 'Bolillo',              'p', 2.50,  4.00,  4, 1, 100),
('75020002', 'Pan de caja Bimbo',    'p', 28.00, 38.00, 4, 4, 20),
('75020003', 'Pan tostado',          'p', 22.00, 30.00, 4, 4, 15),
('75030001', 'Leche Lala 1L',        'p', 18.00, 25.00, 3, 3, 30),
('75030002', 'Yakult',               'p', 10.00, 15.00, 3, 4, 40),
('75030003', 'Queso panela 200g',    'p', 28.00, 38.00, 3, 1, 20),
('75040001', 'Sabritas 35g',         'p', 8.00,  12.00, 7, 4, 80),
('75040002', 'Doritos 50g',          'p', 10.00, 15.00, 7, 4, 60),
('75040003', 'Churrumais 30g',       'p', 7.00,  11.00, 7, 4, 50),
('75050001', 'Jabón Zote 200g',      'p', 14.00, 20.00, 6, 1, 25),
('75050002', 'Cloro 1L',             'p', 12.00, 18.00, 6, 3, 20),
('75050003', 'Fabuloso 1L',          'p', 16.00, 23.00, 6, 3, 20),
('75060001', 'Arroz 1kg',            'p', 18.00, 25.00, 1, 2, 30),
('75060002', 'Frijol 1kg',           'p', 22.00, 30.00, 1, 2, 25),
('75060003', 'Aceite 1L',            'p', 25.00, 35.00, 1, 3, 20),
('75060004', 'Azúcar 1kg',           'p', 20.00, 28.00, 1, 2, 25),
('75060005', 'Sal 1kg',              'p', 8.00,  13.00, 1, 2, 30),
('75070001', 'Shampoo 200ml',        'p', 25.00, 35.00, 5, 1, 15),
('75070002', 'Jabón de baño',        'p', 12.00, 18.00, 5, 1, 30),
('75070003', 'Papel higiénico 4rol', 'p', 22.00, 30.00, 5, 4, 40),
('75080001', 'Huevo 1kg',            'p', 28.00, 38.00, 3, 2, 20),
('75080002', 'Tortillas 1kg',        'p', 15.00, 22.00, 4, 2, 30),
('75080003', 'Pollo 1kg',            'p', 55.00, 75.00, 8, 2, 15);
