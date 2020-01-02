# SWTIS - Study Workflow Tracking Information System

My final project while studying in VIKO EIF (software engineering).

# Purpose

Studying in sessions (every 6 months: 2x2 weeks study sessions and 1x1 week exams session) is not an easy thing to do when you have a full-time job. Sometimes you just can't attend studies session, therefore you are forced to reach teachers via email (and they sometimes don't reply at all). In addition, other students (from the same group) are dealing with the exact same issue. Struggling with this lack of information, assignments tracking is also a tricky thing (what is done, what is not and what needs to be brought to college on what date etc...), so I decided to create the solution on my own - an information system (web application) for studies tracking, which could also be used for other students from the same group.

This web application lets users track the following details:
* Assignments. What needs to be done? Until when? From which subject? Who is the teacher?
* Progress board for assignments (either it's *Not started*, *In progress*, *Pending* or *Completed*). Everyone's personal board with personal note for each assignment.
* Subjects. Who is the teacher? What is subject's website and access key?
* Semesters. Always there 1 and only 1 active semester, so we can easily filter out subjects/assignments by semester, or see only active semester's subjects/assignments.
* Events (like calendar, but in the list format). Things like National exam date?
* Links. Like shared gdrive/onedrive link? College website link?

This application aims to be a "homepage" for students group, managed only by the same students. In addition, there are only 2 types of users (aka roles) - regular users who has view-only permissions (they can still manage their personal progress board) and users with full rights to edit everything (except other people's progress board).

# How to use
## Preparation
1. Install and start MariaDB server. We will configure user and scheme later.

2. Set-up GO programming language. To ensure it is working, try this command:
```
$ go version
go version go1.13.5 linux/amd64
```
Then install required GO build/run dependencies for this project:
```
go get -u github.com/gin-gonic/gin
go get -u github.com/go-sql-driver/mysql
go get -u github.com/gin-contrib/sessions
go get -u golang.org/x/crypto/bcrypt
```
## Installation
1. Compile source code to binary:
```
git clone https://github.com/erkexzcx/Study-Workflow-Tracking-Information-System
cd Study-Workflow-Tracking-Information-System
go build -o swtis src/*.go
```
2. "Install" by copying files to OS locations:
```
mkdir -p /opt/swtis
cp -r {static,templates,swtis.conf,swtis} /opt/swtis/
cp swtis.service /etc/systemd/system/
```
3. Set-up database scheme.

First, create database `bdarbas` with user `bdarbas` and password `bdarbas`. All these settings can be changed in project's file `swtis.conf`.

Then execute SQL file against database to create scheme:
```
mysql -u bdarbas -pbdarbas bdarbas < bdarbas.sql
```
## Usage
There is basic SystemD script included. Usage as any other SystemD service:
```
systemctl <start/stop/enable/disable> swtis.service
```
Then access running project via browser (username `admin` with password `admin`):
```
http://<address>:80
```

# Issues

Thanks to amazing Vilnius College study programme, I **officially** had only **1 month** (I have full-time job as well) to **fully complete** this project (including **fully completed** (minimum 57 pages of) report), so there was not much time to do a research on certain things.

These are the issues I am aware of and I just did not have time to deal with them:

*Server-side*:
1. No tests written.
2. Some repetitive code.
3. User input validation is almost non-existent (project is protected from SQL injections tho).
4. SystemD provided service file is a joke, but it works.
5. `/opt/` should better be replaced with `/etc/`.
6. I should have used a proper ORM. Like [GORM](https://github.com/jinzhu/gorm).
7. There are no soft-deletes. ORM could easily solve this for me.
8. No DB backups implementation.
9. (Error) logging is also very poor.

*Client-side*:
1. Based on AdminLTE2, but AdminLTE3 is already out at the time of writting.
2. Modals, Datatables and some other code is repetitive (even comments - I copy/paste'd a lot!).
