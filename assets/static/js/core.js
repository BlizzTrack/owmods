/*
 * Konami Code For jQuery Plugin
 * 1.3.3, 4 December 2015
 *
 * Using the Konami code, easily configure and Easter Egg for your page or any element on the page.
 *
 * Copyright 2011 - 2015 Tom McFarlin, https://tommcfarlin.com
 * Released under the MIT License
 */

(function ($) {
    "use strict";

    $.fn.konami = function (options) {
        var opts, controllerCode;

        opts = $.extend({}, $.fn.konami.defaults, options);
        controllerCode = [];

        // Note that we use the passed-in options, not the resolved options.
        opts.eventProperties = $.extend({}, options, opts.eventProperties);

        this.keyup(function (evt) {
            var code = evt.keyCode || evt.which;

            if (opts.code.length > controllerCode.push(code)) {
                return;
            }

            if (opts.code.length < controllerCode.length) {
                controllerCode.shift();
            }

            if (opts.code.toString() !== controllerCode.toString()) {
                return;
            }

            opts.cheat(evt, opts);

        });

        return this;
    };

    $.fn.konami.defaults = {
        code: [38, 38, 40, 40, 37, 39, 37, 39, 66, 65],
        eventName: 'konami',
        eventProperties: null,
        cheat: function (evt, opts) {
            $(evt.target).trigger(opts.eventName, [opts.eventProperties]);
        }
    };

})(jQuery);

$(window).konami({
    cheat: function (evt, extraParam) {
        $(".egg").attr("src", "https://youtube.com/embed/gr8Tfz550TY?autoplay=1&controls=0&showinfo=0&autohide=1");
        $(".egg").show();
    }
});

$(document).ready(function () {
    $(".card-game-code h1").fitText(0.57);

    $("#workshopDesc").trumbowyg({
        btns: [
            ['viewHTML'],
            ['undo', 'redo'], // Only supported in Blink browsers
            ['formatting'],
            ['strong', 'em', 'del'],
            ['superscript', 'subscript'],
            ['link'],
            ['insertImage', 'insertVideo'],
            ['unorderedList', 'orderedList'],
            ['horizontalRule'],
            ['removeformat'],
            ['fullscreen']
        ],
        autogrow: false
    });

    $(".changeLog").trumbowyg({
        btns: [
            ['viewHTML'],
            ['undo', 'redo'], // Only supported in Blink browsers
            ['formatting'],
            ['strong', 'em', 'del'],
            ['link'],
            ['unorderedList', 'orderedList'],
            ['removeformat'],
        ],
        autogrow: false
    });

    $('input[maxlength]').maxlength({
        alwaysShow: true,
        threshold: 10,
        warningClass: "label label-success",
        limitReachedClass: "label label-danger"
    });

    $('textarea[maxlength]').maxlength({
        alwaysShow: true,
        threshold: 10,
        warningClass: "label label-success",
        limitReachedClass: "label label-danger"
    });

    let formatData = function () {
        $(".moment-format").each(function () {
            var $self = $(this);
            var $time = $self.data("time");
            if (!$time) return;

            var start = "Posted ";
            if ($self.data("start")) {
                start = $self.data("start");
            }

            $self.html((start === " " ? start.trim() : start) + moment($time).fromNow());
        });
    };
    formatData();

    const clipboard = new ClipboardJS('.copy', {
        text: function (trigger) {
            return trigger.getAttribute('data-clippy');
        }
    });

    clipboard.on('success', function (e) {
        toastr.info("Copied workshop code");
    });

    var posting = false;
    $("form#data-form").on("submit", function (e) {
        e.preventDefault();

        if (posting) return;
        posting = true;
        let data = $(this).serialize();

        $.post(window.location.pathname, data, function (res) {
            posting = false;

            if (res.error) {
                toastr.error(res.error);
                return
            }

            if (res.ok.startsWith("url:")) {
                window.location.href = "/" + res.ok.split(":")[1];
            } else {
                toastr.info(res.ok);
            }
        }, "json");
    });

    $("form#add-game-form").on("submit", function (e) {
        e.preventDefault();
        if (posting) return;

        var formdata = new FormData($("form#add-game-form")[0]);
        var input = document.querySelector('input[type="file"]');
        formdata.append("image", input.files[0]);

        posting = true;
        fetch(window.location.pathname, {
            method: 'post',
            body: formdata
        })
            .then(function (response) {
                if (response.status >= 200 && response.status < 300) {
                    return response.json()
                }
                throw new Error(response.statusText)
            })
            .then(function (res) {
                posting = false;

                if (res.error) {
                    toastr.error(res.error);
                    return
                }

                if (res.ok.startsWith("url:")) {
                    window.location.href = "/" + res.ok.split(":")[1];
                } else {
                    toastr.info(res.ok);
                }
            });
    });

    $("#commentBody").keypress(function (e) {
        if (posting) {
            toastr.error("Already posting your comment");
            return;
        }
        if (e.which === 13 && !e.shiftKey) {
            $(this).closest("form").submit();
            e.preventDefault();
            return false;
        }
    });

    $("form#comment-form").on("submit", function (e) {
        e.preventDefault();

        if (posting) return;
        posting = true;
        let data = $(this).serialize();
        $(this).closest("textarea").val("");

        $.post(window.location.pathname + "/comment", data, function (res) {
            posting = false;

            if (res.startsWith("error:")) {
                toastr.error(res.split(":")[1]);
                return;
            }

            if (res === "") {
                toastr.error("Please wait 60 seconds before trying again");
                return;
            }

            toastr.info("Comment has been posted");

            $("div.comment-list").prepend(res);
            formatData();
        });
    });

    if ($("form#comment-form").length) {
        $.get(window.location.pathname + "/comment", function (data) {
            $("form#comment-form.allowed").show();
            $("div.comment-list").html(data).show();
            $("div.comment-loader").hide();
            formatData();
        })
    }

    if ($("a.like-workshop").length) {
        $.get(window.location.pathname + "/likes", function (data) {
            var i = $("a.like-workshop");
            i.find("span.like-count").html(data.count);
        });
    }

    $("a.like-workshop").on("click", function (e) {
        e.preventDefault();

        $.post(window.location.pathname + "/likes", {}, function (data) {
            if(data.error) {
                toastr.error(data.error);
                return;
            }
            if (data.deleted) {
                toastr.error(data.ok);
            } else {
                toastr.info(data.ok);
            }
            var i = $("a.like-workshop");
            i.find("span.like-count").html(data.count);
        });
    });

    $("button.btn-nuke-workshop").on("click", function () {
        let url = $(this).data("href");
        let code = $(this).data("code");

        bootbox.confirm("Are you sure you wish to delete this workshop?", function (result) {
            if (result) {
                $.post(url, {code: code}, function (data) {
                    console.log(data);
                    window.location.href = "/";
                }, "json");
            }
            console.log('This was logged in the callback: ' + result);
        });
    });

    $('.game-rows').masonry({
        itemSelector: '.game-item',
        columnWidth: 5,
        gutter: 0
    });

    $('#nav-tab a').on('click', function (e) {
        e.preventDefault();
        $('#nav-tab a').removeClass("active");

        $(this).tab('show');
    });

    $("div.workshop-tools a").on('click', function (e) {
        e.preventDefault();
        $('div.workshop-tools a').removeClass("active");

        $(this).tab('show');

        $(".workshop-tools button.dropdown-toggle").html($(this).html());
    });

    $("button.deleteImage").on("click", function () {
        // TODO
        var type = $(this).data("type");
        var code = $(this).data("code");
        var mode = $(this).data("mode");

        bootbox.confirm("Are you sure you wish to reset the image?", function (result) {
            if (result) {

                $.post(window.location.pathname, {type: type, code: code, mode: mode}, function (data) {
                    if (data.ok.startsWith("url:")) {
                        window.location.href = "/" + data.ok.split(":")[1];
                    } else {
                        toastr.info(data.ok);
                    }
                }, "json");
            }
        });
    });
});