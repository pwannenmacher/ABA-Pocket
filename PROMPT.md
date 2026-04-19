# ABA Pocket

Schreibe mir eine Webanwendung mit GoLang im Backend, die zur erfassung und Generierung von druckbaren Taschenkarten im DIN A7 Format dient.
- Beim Ausdruck sollen mehrere der Taschenkarten auf einer DIN A4 Seite zusammengefasst werden.
- Jede Karte muss zwingend die Maße von DIN A7 haben.
- Die DIN A4 Seite mit mehreren Karten soll als PDF exportierbar sein.
- Es gibt 2 Sorten von Taschenkarte. Die erste hat als Überschrift ein Leitsymptom und darunter in 2-Spalten-Tabellenform Medikamente, deren Dosierung und optional einige Zusatzinfos angegeben. Die Zweite sind Medikamentensteckbriefe mit einem Medikamentennamen als Titel und darunter wieder eine 2-Spalten-Tabelle.
- In der Onlineversion sollen Links den schellen Wechsel zwischen Leitsymptomen und Medikamenten erlauben.
- Die Onlineversion soll auch eine Suchfunktion für Leitsymptome und Medikamente enthalten.
- Die Onlineversion soll für eine Darstellung auf einem Smartphone optimiert sein.
- Zu jedem Medikament und Leitsymptom soll ersichtlich sein, wann die letzte Aktualisierung der Informationen stattgefunden hat.
- Zu jedem Medikament und Leitsymptom soll ersichtlich sein, woher die Informationen stammen.
- Die Tabellen der Taschenkarten sollen individuell konfigurierbar sein. Es sollen eigene Schlüsselwörter (linke Spalte) und Werte (rechte Spalte) angegeben werden. In beiden Spalten soll eine Formatierung des Testes (Zeilenumbruch, Aufzählung, Fettschreibung, ...) möglich sein. Evtl. kann Markdown akzeptiert werden.
- Die Daten sollen in einer Datenbank (PostgreSQL) gespeichert werden.
- Die Anwendung soll in Docker Containern gepackt werden.
- Die Datenpflege soll nur durch einen Administrator erfolgen. Dieser soll sich mit einem Benutzernamen und Passwort anmelden können. Es soll eine einfache Benutzerverwaltung geben, um weitere Administratoren hinzufügen zu können.
- Die Anwendung soll im Frontend auf Deutsch Informationen darstellen. 