---
name: tally-erp
description: Skill for cgiving Tally ERP software access to your Claude and other AI models for querying and retriving data from Tally Accounting Software and do processing.
license: MIT
metadata:
  author: Piyush Garg
  version: "1.0.0"
---

# Tally Accounting Software

Tally.ERP 9 is a comprehensive Enterprise Resource Planning software, developed by Tally Solutions, widely used for managing accounting, inventory, taxation (GST), payroll, and compliance in real-time. It simplifies day-to-day business operations, from invoicing to financial reporting, and is favored for its simplicity, speed, and versatility, especially among small-to-medium enterprises.

# Tally API & Integration

TallyPrime has supported integration with web scripting languages such as ASP/Perl/PHP and other languages like VB or any environment capable of supporting XML and HTTP. Integration with these products is possible as XML import and export capability is built into TallyPrime.

## XML Schema Overview and Base Document Structure

TallyPrime uses a consistent XML structure for communication, built around an ENVELOPE element that contains a HEADER and BODY.

Components of Request or Response
<ENVELOPE> is the top element of the XML fragment which is representing the message. Both Request and Response consists of two sections:

Header

Body

Header Information
Header section will give all identification information to the recipient such as authentication, transaction management, and payment so on. This section determines how the recipient of the message should process the information. Header information is classified in two ways, one is for Request and the other is for Response. All the information about Request or Response is enclosed with Header Tags.

In case of Request, header information includes mainly four elements which are Version, TallyRequest, Type and ID. Version gives the version of the message format. Second element TallyRequest will identify the type of request as Import or Export in the messaging format. If the value of Tally Request is Import then the type of information would be Data, and the request will be identified by the report name specified in ID. If the value of Tally Request is Export then the type of information would be Data, Collection, Object or Function. The ID specifies the name of Report, Collection, Object or function.

In the case of Response, there are mainly two elements which are Version and Status. Version gives the version of the message format. Status indicates whether the request is success or failure.

Body Information
It exchanges the information intended for the recipient of the message. This section gives the actual details of the message. It is further divided into two sections:

Description for Request/Response

Data required for the Request/Response

Description section is used to give the description for message, request or response. Description element mainly includes all types of variable information, storage information, computational information and user defined TDLs. All the description information is enclosed with <DESC> tags.

Data section includes all the data information being transferred. All the data should be enclosed within the <DATA> tags.

Basic Template:

```xml
<ENVELOPE>

<HEADER>

<TALLYREQUEST>Import Data</TALLYREQUEST>

</HEADER>

<BODY>

<IMPORTDATA>

<REQUESTDESC>

<REPORTNAME>Vouchers</REPORTNAME>

</REQUESTDESC>

<REQUESTDATA>

<TALLYMESSAGE>

<!– XML data for vouchers, ledgers, etc. –>

</TALLYMESSAGE>

</REQUESTDATA>
</IMPORTDATA>
</BODY>
</ENVELOPE>
```
