$(document).ready(function () {

    // Toastr notification settings:
    toastr.options = {
        "closeButton": false,
        "debug": false,
        "newestOnTop": false,
        "progressBar": true,
        "positionClass": "toast-top-right",
        "preventDuplicates": false,
        "onclick": null,
        "showDuration": "300",
        "hideDuration": "1000",
        "timeOut": "5000",
        "extendedTimeOut": "1000",
        "showEasing": "swing",
        "hideEasing": "linear",
        "showMethod": "fadeIn",
        "hideMethod": "fadeOut"
    }

    // Load active subjects:
    refreshActiveSubjectsContainer()

    // Load links:
    refreshLinksContainer();

    // Load labels:
    refreshMenuLabels();

    // Sidebar menu workarounds for ajax navigation:
    $('ul.sidebar-menu > li:not(.header):not(#activeSubjectsContainer)').on('click', function () {
        $(this).addClass('active');
        $('.sidebar-open').removeClass('sidebar-open'); // Close sidebar in mobile view
    });
    $('#activeSubjectsContainer').prop("onclick", null).off("click"); // Remove onclick events, set by AdminLTE code

    // Initialize onclick events for elements with ajax links:
    $('[ajaxclickable]').on('click', function () {
        Pace.restart(); // Show progress bar only when clicking on such element and NOT with EVERY ajax request.
        $('ul.sidebar-menu > li.active').removeClass('active');
        loadPage('/' + $(this).attr('href').substring(1));
    });

    // Also load something on website's load:
    if (window.location.hash) {
        $('[href="' + window.location.hash + '"]:first').click();
    } else {
        $('[href="#progress_board"]:first').click();
    }

    // Create few custom rules for JQuery validator:
    jQuery.validator.addMethod("atLeastUppercase", function (value, element) {
        if (value == '') {
            return true;
        } else {
            return /[A-Z]/.test(value);
        }
    }, 'Must contain at least one UPPERCASE letter');

    jQuery.validator.addMethod("atLeastLowercase", function (value, element) {
        if (value == '') {
            return true;
        } else {
            return /[a-z]/.test(value);
        }
    }, 'Must contain at least one lowercase letter');

    jQuery.validator.addMethod("atLeastNumber", function (value, element) {
        if (value == '') {
            return true;
        } else {
            return /[0-9]/.test(value);
        }
    }, 'Must contain at least one number');

    jQuery.validator.addMethod("nothingOrMinimum", function (value, element) {
        if (value == '') {
            return true;
        }
        if (value.length > 0 && value.length < 6) {
            return false;
        }
        return true;
    }, 'Must be nothing or more than 6 characters long!');

    jQuery.validator.addMethod("noSpace", function(value, element) { 
        return value.indexOf(" ") < 0; 
    }, "Must not contain spaces!");

    jQuery.validator.addMethod("notEqualTo", function (value, element, param) {
        return this.optional(element) || value != $(param).val();
    }, 'Must not be the same as your old password');

});

// Used when need to show "loading" inside the modal before getting JSON content via AJAX:
function showModalContentLoading(modal) {
    modal.find('.modal-title').html('Loading...');
    modal.find('.modal-body.loaded').hide();
    modal.find('.modal-body.loading').show();
    modal.find('.modal-body.error').hide();
}
function showModalContentLoaded(modal) {
    modal.find('.modal-body.loaded').show();
    modal.find('.modal-body.loading').hide();
    modal.find('.modal-body.error').hide();
}
function showModalContentFailed(modal) {
    modal.find('.modal-body.loaded').hide();
    modal.find('.modal-body.loading').hide();
    modal.find('.modal-body.error').show();
}

// Call this function to refresh active subjects container in app.html left-side menu
function refreshActiveSubjectsContainer() {

    $.ajax({
        type: 'GET',
        url: '/app/activeSubjectsContainer',
        dataType: 'json',
        success: function (data) {
            var container = $('#activeSubjectsContainer');
            container.html('');
            if (data) {
                $('.activeSubjectsAdditionalElements').show()
                for (var i = 0; i < data.length; i++) {
                    if (data[i].url == "") {
                        container.append('<a class="addedActiveSubjects" target="_blank" style="color: #8c8c8c; font-style: italic;"><span>' + data[i].title.substring(0, 33) + '</span>');
                    } else {
                        container.append('<a class="addedActiveSubjects" target="_blank" href="' + data[i].url + '"><span>' + data[i].title.substring(0, 33) + '</span>');
                    }
                }
            } else {
                $('.activeSubjectsAdditionalElements').hide()
            }
        },
        error: ajaxErrorHandler
    });

}

// Call this function to refresh links container in app.html top-right menu dropdown
function refreshLinksContainer() {

    $.ajax({
        type: 'GET',
        url: '/app/linksContainer',
        dataType: 'json',
        success: function (data) {
            var container = $('#links_container');
            container.html('');
            if (data){
                for (var i = 0; i < data.length; i++) {
                    container.append('<li><a style="cursor: pointer;" onclick="openInNewTab(\'' + data[i].url + '\');">' + data[i].title + '</a></li>');
                }
            }else{
                container.append('<li>No links...</li>')
            }

        },
        error: ajaxErrorHandler
    });

}

// Call this function to refresh menu labels (events, assignments) in app.html left-side menu
function refreshMenuLabels() {

    $.ajax({
        type: 'GET',
        url: '/app/menuLabels',
        dataType: 'json',
        success: function (data) {
            var eventsContainer = $('#events_labels_container');
            var assignmentsContainer = $('#assignments_labels_container');

            eventsContainer.html('');
            assignmentsContainer.html('');

            if (data.events_yellow > 0) {
                eventsContainer.append('<small class="label pull-right label-warning">' + data.events_yellow + '</small>');
            }
            if (data.events_red > 0) {
                eventsContainer.append('<small class="label pull-right label-danger">' + data.events_red + '</small>');
            }

            if (data.assignments_yellow > 0) {
                assignmentsContainer.append('<small class="label pull-right label-warning">' + data.assignments_yellow + '</small>');
            }
            if (data.assignments_red > 0) {
                assignmentsContainer.append('<small class="label pull-right label-danger">' + data.assignments_red + '</small>');
            }

            // Add some weird animation :)
            setTimeout(function () {
                eventsContainer.fadeOut(100).fadeIn(100).fadeOut(100).fadeIn(100);
            }, 0)
            setTimeout(function () {
                assignmentsContainer.fadeOut(100).fadeIn(100).fadeOut(100).fadeIn(100);
            }, 0)
        },
        error: ajaxErrorHandler
    });

}

// Executed from jquery validator submithandler:
function submitAjaxForm(form, successMessage, successCallback) {

    // Disable submit button:
    $(form).find('[type="submit"]').addClass('disabled');

    // Submit data to server:
    $.ajax({
        type: $(form).attr('method'), // form.method returns GET if you set to something else rather than POST/GET
        url: form.action,
        data: $(form).serialize(),
        dataType: 'json',
        success: function (data) {
            toastr.success(successMessage);
            successCallback();
        },
        error: function (request, status, error) {
            ajaxErrorHandler(request, status, error);
            $(form).find('[type="submit"]').removeClass('disabled');
        }
    });
}

// Use this function to load page to main website's container::
function loadPage(page) {
    var content = $('#load-here');

    // Hide existing content
    content.fadeOut();

    // Submit data to server:
    $.ajax({
        type: 'GET',
        url: page,
        success: function (data) {
            // Wait for "fadeOut" to finish:
            content.promise().done(function () {
                // Fade In (speaking programatically, HTML becomds visible instantly after this command):
                content.html(data).fadeIn('fast');
                // So when content is visible, do what is needed:
                postPageLoading();
            });
        },
        error: ajaxErrorHandler
    });
}

// Execute every time after loading new page into main website's container:
function postPageLoading() {
    content = $('#load-here');

    //==============================================================

    // Beautiful checkboxes and radio buttons:
    content.find('input').iCheck({
        checkboxClass: 'icheckbox_square-blue',
        radioClass: 'iradio_square-blue'
    });

    //==============================================================

    // Do not allow these textareas go multiline (because new line (\n) is considered as character in PHP & SQL, but not in javascript):
    content.find('.single-line-textarea').on("change", function () {
        text = $(this).val().trim();
        $(this).val(text.replace(/\n/g, " "));
    });

    //==============================================================

    // Initialize datepicker:
    content.find('.datepicker').datepicker({
        autoclose: true,
        todayHighlight: true,
        format: 'yyyy-mm-dd',
        weekStart: 1 // day of the week start. 0 for Sunday - 6 for Saturday
    });

    //==============================================================

    // Modals

    // Add html code to each modal for later use:
    content.find('.ajax-content > .modal-body.loaded').after(`

        <div class="modal-body loading" style="display: none">
            <p><i class="fa fa-spinner fa-spin" aria-hidden="true"></i> Loading...</p>
        </div>

        <div class="modal-body error" style="display: none">
            <p style="color: #FFE000"><i class="fa fa-times-circle-o"></i> Failed to load data from the
                server. Please close this dialog and try again.</p>
        </div>

    `);

    // Hide all modals:
    content.find('.modal.in').modal('hide');
    $('body').removeClass('modal-open');
    $('.modal-backdrop').remove();
    $('body').attr('style', '');

    //==============================================================

    // Initialize "google keep" like grid:
    content.find('.grid').packery({
        // use a separate class for itemSelector, other than .col-
        itemSelector: '.grid-item',
        columnWidth: '.grid-sizer',
        percentPosition: true
    });

    //==============================================================

    // Initialise on click events for content after loading it:
    content.find('[href^="#"]:not([href="#"]):not([href^="#tab_"])').on('click', function () {
        Pace.restart(); // Show progress bar only when clicking on such element and NOT with EVERY ajax request.
        loadPage('/pages/' + $(this).attr('href').substring(1));
    });

    //==============================================================

    // JQuery input mask plugin:
    // https://github.com/RobinHerbots/Inputmask
    content.find("input").inputmask();

    //==============================================================

    // Recalculate DataTables (this fixes responsiveness when HTML loaded to hidden element):
    var table = $('.dataTables_wrapper > table').DataTable();
    table.responsive.recalc();
}

// Handle errors that might come from ajax requests.
function ajaxErrorHandler(request, status, error) {
    var msg = "Unknown error occurred";
    if (request.status == 0){
        msg = "Error: Server is unreachable";
    }else if (typeof request.responseJSON.message === 'string'){
        msg = "Error: " + request.responseJSON.message;
    }else{
        msg = "Error: " + request.responseText;
    }
    toastr.error(msg);
}

function openInNewTab(url) {
    var win = window.open(url, '_blank');
    win.focus();
  }